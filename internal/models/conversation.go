package models

import (
	"time"
)

type Conversation struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Tool      string    `json:"tool"`
	Project   string    `json:"project"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages,omitempty"`
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

type ConversationStats struct {
	TotalConversations int     `json:"total_conversations"`
	TotalMessages      int     `json:"total_messages"`
	TotalTokens        int     `json:"total_tokens"`
	EstimatedCost      float64 `json:"estimated_cost"`
	ToolBreakdown      map[string]int `json:"tool_breakdown"`
	ProjectBreakdown   map[string]int `json:"project_breakdown"`
}