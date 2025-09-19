package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jasper/ai-memory/internal/models"
)

type SQLiteStore struct {
	db *sql.DB
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

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			tool TEXT NOT NULL,
			project TEXT,
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			token_count INTEGER DEFAULT 0,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_tool ON conversations(tool)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_project ON conversations(project)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_created ON conversations(created_at)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
			content,
			content=messages,
			content_rowid=id
		)`,
		`CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages
		BEGIN
			INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages
		BEGIN
			DELETE FROM messages_fts WHERE rowid = old.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages
		BEGIN
			UPDATE messages_fts SET content = new.content WHERE rowid = new.id;
		END`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) SaveConversation(conv *models.Conversation) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tagsJSON, _ := json.Marshal(conv.Tags)

	result, err := tx.Exec(
		`INSERT INTO conversations (title, tool, project, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		conv.Title, conv.Tool, conv.Project, string(tagsJSON), conv.CreatedAt, conv.UpdatedAt,
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
			`INSERT INTO messages (conversation_id, role, content, timestamp, token_count)
			VALUES (?, ?, ?, ?, ?)`,
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

func (s *SQLiteStore) GetConversation(id int64) (*models.Conversation, error) {
	conv := &models.Conversation{}
	var tagsJSON string

	err := s.db.QueryRow(
		`SELECT id, title, tool, project, tags, created_at, updated_at
		FROM conversations WHERE id = ?`, id,
	).Scan(&conv.ID, &conv.Title, &conv.Tool, &conv.Project, &tagsJSON, &conv.CreatedAt, &conv.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &conv.Tags)
	}

	rows, err := s.db.Query(
		`SELECT id, conversation_id, role, content, timestamp, token_count
		FROM messages WHERE conversation_id = ? ORDER BY timestamp`, id,
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

	rows, err := s.db.Query(query, args...)
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
	searchQuery := `
		SELECT DISTINCT
			c.id, c.title, c.tool, c.project, c.tags, c.created_at, c.updated_at,
			m.content, bm25(messages_fts) as score
		FROM messages_fts
		JOIN messages m ON messages_fts.rowid = m.id
		JOIN conversations c ON m.conversation_id = c.id
		WHERE messages_fts MATCH ?
		ORDER BY score DESC
		LIMIT ?
	`

	rows, err := s.db.Query(searchQuery, query, limit)
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

	err := s.db.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&stats.TotalConversations)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&stats.TotalMessages)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRow(`SELECT COALESCE(SUM(token_count), 0) FROM messages`).Scan(&stats.TotalTokens)
	if err != nil {
		return nil, err
	}

	stats.EstimatedCost = float64(stats.TotalTokens) * 0.000003

	rows, err := s.db.Query(`SELECT tool, COUNT(*) FROM conversations GROUP BY tool`)
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

	rows, err = s.db.Query(`SELECT project, COUNT(*) FROM conversations WHERE project IS NOT NULL GROUP BY project`)
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
	_, err := s.db.Exec(`DELETE FROM conversations WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
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

	_, err := s.db.Exec(
		`UPDATE conversations SET title = ?, tool = ?, project = ?, tags = ?, updated_at = ? WHERE id = ?`,
		conv.Title, conv.Tool, conv.Project, string(tagsJSON), conv.UpdatedAt, conv.ID,
	)
	return err
}