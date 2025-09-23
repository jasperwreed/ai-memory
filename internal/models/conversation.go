package models

import (
	"time"
)

type Conversation struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Tool        string    `json:"tool"`
	Project     string    `json:"project"`
	ProjectID   int64     `json:"project_id,omitempty"`
	ProjectPath string    `json:"project_path,omitempty"`
	Tags        []string  `json:"tags"`
	SessionID   string    `json:"session_id,omitempty"`
	SourcePath  string    `json:"source_path,omitempty"`
	AuditShard  string    `json:"audit_shard,omitempty"`
	RawJSON     string    `json:"-"` // Don't include in JSON output
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Messages    []Message `json:"messages,omitempty"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
	TokenCount     int       `json:"token_count,omitempty"`
}

type SearchResult struct {
	Conversation Conversation `json:"conversation"`
	Snippet      string       `json:"snippet"`
	Score        float64      `json:"score"`
}

type Project struct {
	ID          int64  `json:"id"`
	ProjectPath string `json:"project_path"`
}

type ConversationStats struct {
	TotalConversations int     `json:"total_conversations"`
	TotalMessages      int     `json:"total_messages"`
	TotalTokens        int     `json:"total_tokens"`
	EstimatedCost      float64 `json:"estimated_cost"`
	ToolBreakdown      map[string]int `json:"tool_breakdown"`
	ProjectBreakdown   map[string]int `json:"project_breakdown"`
}