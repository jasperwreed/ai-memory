package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Handlers struct {
	db *sql.DB
}

func NewHandlers(db *sql.DB) *Handlers {
	return &Handlers{db: db}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handlers) ListConversations(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	limit := 50
	if l := queryParams.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := queryParams.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query with optional filters
	query := `
		SELECT
			id, tool, project, session_id, source_file,
			summary, token_count, created_at, updated_at
		FROM conversations
		WHERE 1=1`

	args := []interface{}{}

	if tool := queryParams.Get("tool"); tool != "" {
		query += " AND tool = ?"
		args = append(args, tool)
	}

	if project := queryParams.Get("project"); project != "" {
		query += " AND project = ?"
		args = append(args, project)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to query conversations")
		return
	}
	defer rows.Close()

	conversations := []map[string]interface{}{}
	for rows.Next() {
		var conv struct {
			ID         int64
			Tool       sql.NullString
			Project    sql.NullString
			SessionID  sql.NullString
			SourceFile sql.NullString
			Summary    sql.NullString
			TokenCount int
			CreatedAt  time.Time
			UpdatedAt  time.Time
		}

		err := rows.Scan(
			&conv.ID, &conv.Tool, &conv.Project, &conv.SessionID,
			&conv.SourceFile, &conv.Summary, &conv.TokenCount,
			&conv.CreatedAt, &conv.UpdatedAt,
		)
		if err != nil {
			continue
		}

		conversations = append(conversations, map[string]interface{}{
			"id":          conv.ID,
			"tool":        conv.Tool.String,
			"project":     conv.Project.String,
			"session_id":  conv.SessionID.String,
			"source_file": conv.SourceFile.String,
			"summary":     conv.Summary.String,
			"token_count": conv.TokenCount,
			"created_at":  conv.CreatedAt.Format(time.RFC3339),
			"updated_at":  conv.UpdatedAt.Format(time.RFC3339),
		})
	}

	response := map[string]interface{}{
		"conversations": conversations,
		"limit":         limit,
		"offset":        offset,
		"total":         len(conversations),
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handlers) GetConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	query := `
		SELECT
			id, tool, project, session_id, source_file,
			summary, token_count, created_at, updated_at
		FROM conversations
		WHERE id = ?
	`

	var conv struct {
		ID         int64
		Tool       sql.NullString
		Project    sql.NullString
		SessionID  sql.NullString
		SourceFile sql.NullString
		Summary    sql.NullString
		TokenCount int
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}

	err := h.db.QueryRow(query, id).Scan(
		&conv.ID, &conv.Tool, &conv.Project, &conv.SessionID,
		&conv.SourceFile, &conv.Summary, &conv.TokenCount,
		&conv.CreatedAt, &conv.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get conversation")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":          conv.ID,
		"tool":        conv.Tool.String,
		"project":     conv.Project.String,
		"session_id":  conv.SessionID.String,
		"source_file": conv.SourceFile.String,
		"summary":     conv.Summary.String,
		"token_count": conv.TokenCount,
		"created_at":  conv.CreatedAt.Format(time.RFC3339),
		"updated_at":  conv.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Handlers) CreateConversation(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Tool     string                   `json:"tool"`
		Project  string                   `json:"project"`
		Summary  string                   `json:"summary"`
		Messages []map[string]interface{} `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(payload.Messages) == 0 {
		respondWithError(w, http.StatusBadRequest, "Messages are required")
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Insert conversation
	result, err := tx.Exec(`
		INSERT INTO conversations (tool, project, summary, token_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, payload.Tool, payload.Project, payload.Summary, 0, time.Now(), time.Now())

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create conversation")
		return
	}

	convID, err := result.LastInsertId()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get conversation ID")
		return
	}

	// Insert messages
	for _, msg := range payload.Messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		_, err = tx.Exec(`
			INSERT INTO messages (conversation_id, role, content, created_at)
			VALUES (?, ?, ?, ?)
		`, convID, role, content, time.Now())

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to insert message")
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         convID,
		"tool":       payload.Tool,
		"project":    payload.Project,
		"summary":    payload.Summary,
		"created_at": time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Delete messages first (foreign key constraint)
	_, err = tx.Exec("DELETE FROM messages WHERE conversation_id = ?", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete messages")
		return
	}

	// Delete conversation
	result, err := tx.Exec("DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete conversation")
		return
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	response := map[string]string{
		"message": "Conversation deleted successfully",
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		respondWithError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	query := `
		SELECT
			c.id, c.tool, c.project, c.summary, c.token_count,
			c.created_at, snippet(conversations_fts, -1, '**', '**', '...', 32) as snippet
		FROM conversations c
		JOIN conversations_fts ON c.id = conversations_fts.rowid
		WHERE conversations_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := h.db.Query(query, searchQuery, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Search failed")
		return
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var result struct {
			ID         int64
			Tool       sql.NullString
			Project    sql.NullString
			Summary    sql.NullString
			TokenCount int
			CreatedAt  time.Time
			Snippet    string
		}

		err := rows.Scan(
			&result.ID, &result.Tool, &result.Project,
			&result.Summary, &result.TokenCount,
			&result.CreatedAt, &result.Snippet,
		)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":          result.ID,
			"tool":        result.Tool.String,
			"project":     result.Project.String,
			"summary":     result.Summary.String,
			"token_count": result.TokenCount,
			"created_at":  result.CreatedAt.Format(time.RFC3339),
			"snippet":     result.Snippet,
		})
	}

	response := map[string]interface{}{
		"query":   searchQuery,
		"results": results,
		"count":   len(results),
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handlers) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{}

	// Total conversations
	var totalConversations int
	h.db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&totalConversations)
	stats["total_conversations"] = totalConversations

	// Total messages
	var totalMessages int
	h.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&totalMessages)
	stats["total_messages"] = totalMessages

	// Total tokens
	var totalTokens int64
	h.db.QueryRow("SELECT COALESCE(SUM(token_count), 0) FROM conversations").Scan(&totalTokens)
	stats["total_tokens"] = totalTokens

	// Tools breakdown
	toolsQuery := `
		SELECT tool, COUNT(*) as count
		FROM conversations
		WHERE tool IS NOT NULL
		GROUP BY tool
		ORDER BY count DESC
	`

	rows, err := h.db.Query(toolsQuery)
	if err == nil {
		defer rows.Close()
		tools := []map[string]interface{}{}
		for rows.Next() {
			var tool string
			var count int
			rows.Scan(&tool, &count)
			tools = append(tools, map[string]interface{}{
				"tool":  tool,
				"count": count,
			})
		}
		stats["tools"] = tools
	}

	// Projects breakdown
	projectsQuery := `
		SELECT project, COUNT(*) as count
		FROM conversations
		WHERE project IS NOT NULL
		GROUP BY project
		ORDER BY count DESC
		LIMIT 10
	`

	rows, err = h.db.Query(projectsQuery)
	if err == nil {
		defer rows.Close()
		projects := []map[string]interface{}{}
		for rows.Next() {
			var project string
			var count int
			rows.Scan(&project, &count)
			projects = append(projects, map[string]interface{}{
				"project": project,
				"count":   count,
			})
		}
		stats["top_projects"] = projects
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func (h *Handlers) GetMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	query := `
		SELECT id, conversation_id, role, content, created_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`

	rows, err := h.db.Query(query, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get messages")
		return
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var msg struct {
			ID             int64
			ConversationID int64
			Role           string
			Content        string
			CreatedAt      time.Time
		}

		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			continue
		}

		messages = append(messages, map[string]interface{}{
			"id":              msg.ID,
			"conversation_id": msg.ConversationID,
			"role":            msg.Role,
			"content":         msg.Content,
			"created_at":      msg.CreatedAt.Format(time.RFC3339),
		})
	}

	response := map[string]interface{}{
		"conversation_id": id,
		"messages":        messages,
		"count":           len(messages),
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (h *Handlers) ExportConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Get conversation details
	convQuery := `
		SELECT id, tool, project, summary, token_count, created_at, updated_at
		FROM conversations
		WHERE id = ?
	`

	var conv struct {
		ID         int64
		Tool       sql.NullString
		Project    sql.NullString
		Summary    sql.NullString
		TokenCount int
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}

	err := h.db.QueryRow(convQuery, id).Scan(
		&conv.ID, &conv.Tool, &conv.Project,
		&conv.Summary, &conv.TokenCount,
		&conv.CreatedAt, &conv.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Conversation not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get conversation")
		return
	}

	// Get messages
	msgQuery := `
		SELECT role, content, created_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`

	rows, err := h.db.Query(msgQuery, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get messages")
		return
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var role, content string
		var createdAt time.Time
		rows.Scan(&role, &content, &createdAt)
		messages = append(messages, map[string]interface{}{
			"role":       role,
			"content":    content,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}

	// Build export data
	exportData := map[string]interface{}{
		"id":          conv.ID,
		"tool":        conv.Tool.String,
		"project":     conv.Project.String,
		"summary":     conv.Summary.String,
		"token_count": conv.TokenCount,
		"created_at":  conv.CreatedAt.Format(time.RFC3339),
		"updated_at":  conv.UpdatedAt.Format(time.RFC3339),
		"messages":    messages,
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"conversation_%s.json\"", id))

	json.NewEncoder(w).Encode(exportData)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}