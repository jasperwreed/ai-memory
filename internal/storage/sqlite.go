package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"github.com/jasperwreed/ai-memory/internal/models"
)

type SQLiteStore struct {
	writeDB *sql.DB  // Single connection for writes
	readDB  *sql.DB  // Pool of connections for reads
	dbPath  string
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".ai-memory", "conversations.db")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open write connection (single connection)
	writeDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open write database: %w", err)
	}
	writeDB.SetMaxOpenConns(1) // Only one write connection

	// Open read connection pool
	// Note: We don't use ?mode=ro here because the database might not exist yet
	// and we need at least one connection to create tables
	readDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("failed to open read database: %w", err)
	}
	readDB.SetMaxOpenConns(5)  // Allow multiple concurrent reads
	readDB.SetMaxIdleConns(5)

	store := &SQLiteStore{
		writeDB: writeDB,
		readDB:  readDB,
		dbPath:  dbPath,
	}

	// Initialize database with optimizations
	if err := store.initializeDB(); err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := store.createTables(); err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) initializeDB() error {
	// Apply SQLite optimizations for performance
	config := DefaultConfig()
	pragmas := config.pragmas()

	for _, pragma := range pragmas {
		if _, err := s.writeDB.Exec(pragma); err != nil {
			return fmt.Errorf("failed to set %s: %w", pragma, err)
		}
	}

	return nil
}

func (s *SQLiteStore) createTables() error {
	queries := []string{
		queryCreateProjectsTable,
		queryCreateConversationsTable,
		queryCreateMessagesTable,
		queryCreateIndexMessagesConversation,
		queryCreateIndexConversationsTool,
		queryCreateIndexConversationsProject,
		queryCreateIndexConversationsProjectID,
		queryCreateIndexConversationsCreated,
		queryCreateIndexConversationsSession,
		queryCreateIndexConversationsSource,
		queryCreateMessagesFTS,
		queryCreateMessagesInsertTrigger,
		queryCreateMessagesDeleteTrigger,
		queryCreateMessagesUpdateTrigger,
	}

	for _, query := range queries {
		if _, err := s.writeDB.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) SaveConversation(conv *models.Conversation) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Handle project insertion/lookup
	var projectID *int64
	if conv.ProjectPath != "" {
		// Insert project if it doesn't exist
		if _, err := tx.Exec(queryInsertProject, conv.ProjectPath); err != nil {
			return fmt.Errorf("failed to insert project: %w", err)
		}

		// Get project ID
		var pid int64
		if err := tx.QueryRow(querySelectProjectID, conv.ProjectPath).Scan(&pid); err != nil {
			return fmt.Errorf("failed to get project ID: %w", err)
		}
		projectID = &pid
		conv.ProjectID = pid
	}

	tagsJSON, _ := json.Marshal(conv.Tags)

	result, err := tx.Exec(
		queryInsertConversation,
		conv.Title, conv.Tool, conv.Project, projectID, string(tagsJSON),
		conv.SessionID, conv.SourcePath, conv.AuditShard, conv.RawJSON,
		conv.CreatedAt, conv.UpdatedAt,
	)
	if err != nil {
		return err
	}

	convID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	conv.ID = convID

	for i := range conv.Messages {
		result, err := tx.Exec(
			queryInsertMessage,
			convID, conv.Messages[i].Role, conv.Messages[i].Content,
			conv.Messages[i].Timestamp, conv.Messages[i].TokenCount,
		)
		if err != nil {
			return err
		}
		msgID, _ := result.LastInsertId()
		conv.Messages[i].ID = msgID
		conv.Messages[i].ConversationID = convID
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetConversationBySessionID(sessionID string) (*models.Conversation, error) {
	query := `SELECT id, title, tool, project, project_id, tags, session_id, source_path, audit_shard, raw_json, created_at, updated_at
		FROM conversations WHERE session_id = ?`

	conv := &models.Conversation{}
	var tagsJSON string
	var projectID sql.NullInt64
	var sessionIDVal, sourcePath, auditShard, rawJSON sql.NullString

	err := s.readDB.QueryRow(query, sessionID).Scan(
		&conv.ID, &conv.Title, &conv.Tool, &conv.Project, &projectID, &tagsJSON,
		&sessionIDVal, &sourcePath, &auditShard, &rawJSON,
		&conv.CreatedAt, &conv.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if projectID.Valid {
		conv.ProjectID = projectID.Int64
	}
	conv.SessionID = sessionIDVal.String
	conv.SourcePath = sourcePath.String
	conv.AuditShard = auditShard.String
	conv.RawJSON = rawJSON.String

	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &conv.Tags)
	}

	return conv, nil
}

func (s *SQLiteStore) GetConversation(id int64) (*models.Conversation, error) {
	conv := &models.Conversation{}
	var tagsJSON string
	var projectID sql.NullInt64
	var sessionID, sourcePath, auditShard, rawJSON sql.NullString

	err := s.readDB.QueryRow(
		querySelectConversation, id,
	).Scan(&conv.ID, &conv.Title, &conv.Tool, &conv.Project, &projectID, &tagsJSON,
		&sessionID, &sourcePath, &auditShard, &rawJSON,
		&conv.CreatedAt, &conv.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if projectID.Valid {
		conv.ProjectID = projectID.Int64
	}
	conv.SessionID = sessionID.String
	conv.SourcePath = sourcePath.String
	conv.AuditShard = auditShard.String
	conv.RawJSON = rawJSON.String

	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &conv.Tags)
	}

	rows, err := s.readDB.Query(
		querySelectMessages, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conv.Messages = []models.Message{}
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Timestamp, &msg.TokenCount)
		if err != nil {
			return nil, err
		}
		conv.Messages = append(conv.Messages, msg)
	}

	return conv, nil
}

func (s *SQLiteStore) ListConversations(limit, offset int, filter map[string]string) ([]models.Conversation, error) {
	query := `SELECT id, title, tool, project, tags, created_at, updated_at FROM conversations WHERE 1=1`
	args := []interface{}{}

	if tool, ok := filter["tool"]; ok {
		query += " AND tool = ?"
		args = append(args, tool)
	}
	if project, ok := filter["project"]; ok {
		query += " AND project = ?"
		args = append(args, project)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.writeDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []models.Conversation
	for rows.Next() {
		var conv models.Conversation
		var tagsJSON string
		err := rows.Scan(&conv.ID, &conv.Title, &conv.Tool, &conv.Project, &tagsJSON, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &conv.Tags)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

func (s *SQLiteStore) Search(query string, limit int) ([]models.SearchResult, error) {
	rows, err := s.readDB.Query(querySearchConversations, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var result models.SearchResult
		var tagsJSON string
		var content string

		err := rows.Scan(
			&result.Conversation.ID, &result.Conversation.Title,
			&result.Conversation.Tool, &result.Conversation.Project,
			&tagsJSON, &result.Conversation.CreatedAt,
			&result.Conversation.UpdatedAt, &content, &result.Score,
		)
		if err != nil {
			return nil, err
		}

		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &result.Conversation.Tags)
		}

		result.Snippet = truncateContent(content, 200)
		results = append(results, result)
	}

	return results, nil
}

func (s *SQLiteStore) GetStats() (*models.ConversationStats, error) {
	stats := &models.ConversationStats{
		ToolBreakdown:    make(map[string]int),
		ProjectBreakdown: make(map[string]int),
	}

	err := s.readDB.QueryRow(queryCountConversations).Scan(&stats.TotalConversations)
	if err != nil {
		return nil, err
	}

	err = s.readDB.QueryRow(queryCountMessages).Scan(&stats.TotalMessages)
	if err != nil {
		return nil, err
	}

	err = s.readDB.QueryRow(querySumTokens).Scan(&stats.TotalTokens)
	if err != nil {
		return nil, err
	}

	stats.EstimatedCost = float64(stats.TotalTokens) * 0.000003

	rows, err := s.readDB.Query(queryGroupByTool)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tool string
		var count int
		if err := rows.Scan(&tool, &count); err == nil {
			stats.ToolBreakdown[tool] = count
		}
	}

	rows, err = s.readDB.Query(queryGroupByProject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var project string
		var count int
		if err := rows.Scan(&project, &count); err == nil {
			stats.ProjectBreakdown[project] = count
		}
	}

	return stats, nil
}

func (s *SQLiteStore) DeleteConversation(id int64) error {
	_, err := s.writeDB.Exec(queryDeleteConversation, id)
	return err
}

func (s *SQLiteStore) Close() error {
	var errs []error

	// Run PRAGMA optimize before closing for better long-term performance
	if _, err := s.writeDB.Exec("PRAGMA optimize"); err != nil {
		errs = append(errs, fmt.Errorf("failed to optimize: %w", err))
	}

	if err := s.readDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close read db: %w", err))
	}

	if err := s.writeDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close write db: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return strings.TrimSpace(content[:maxLen]) + "..."
}

func (s *SQLiteStore) UpdateConversation(conv *models.Conversation) error {
	conv.UpdatedAt = time.Now()
	tagsJSON, _ := json.Marshal(conv.Tags)

	_, err := s.writeDB.Exec(
		`UPDATE conversations SET title = ?, tool = ?, project = ?, tags = ?, updated_at = ? WHERE id = ?`,
		conv.Title, conv.Tool, conv.Project, string(tagsJSON), conv.UpdatedAt, conv.ID,
	)
	return err
}