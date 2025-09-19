package storage

import (
	"os"
	"testing"
	"time"

	"github.com/jasper/ai-memory/internal/models"
)

func TestSQLiteStore(t *testing.T) {
	tmpFile := "/tmp/test_ai_memory.db"
	defer os.Remove(tmpFile)

	store, err := NewSQLiteStore(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("SaveAndGetConversation", func(t *testing.T) {
		conv := &models.Conversation{
			Title:     "Test Conversation",
			Tool:      "test-tool",
			Project:   "test-project",
			Tags:      []string{"tag1", "tag2"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Messages: []models.Message{
				{
					Role:       "user",
					Content:    "Test question",
					Timestamp:  time.Now(),
					TokenCount: 10,
				},
				{
					Role:       "assistant",
					Content:    "Test answer",
					Timestamp:  time.Now(),
					TokenCount: 15,
				},
			},
		}

		err := store.SaveConversation(conv)
		if err != nil {
			t.Fatalf("Failed to save conversation: %v", err)
		}

		if conv.ID == 0 {
			t.Error("Conversation ID should be set after save")
		}

		retrieved, err := store.GetConversation(conv.ID)
		if err != nil {
			t.Fatalf("Failed to get conversation: %v", err)
		}

		if retrieved.Title != conv.Title {
			t.Errorf("Title mismatch: got %s, want %s", retrieved.Title, conv.Title)
		}

		if len(retrieved.Messages) != len(conv.Messages) {
			t.Errorf("Message count mismatch: got %d, want %d",
				len(retrieved.Messages), len(conv.Messages))
		}
	})

	t.Run("ListConversations", func(t *testing.T) {
		conversations, err := store.ListConversations(10, 0, nil)
		if err != nil {
			t.Fatalf("Failed to list conversations: %v", err)
		}

		if len(conversations) == 0 {
			t.Error("Should have at least one conversation")
		}
	})

	t.Run("Search", func(t *testing.T) {
		results, err := store.Search("test", 10)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if len(results) == 0 {
			t.Error("Should find at least one result for 'test'")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats, err := store.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats.TotalConversations == 0 {
			t.Error("Should have at least one conversation in stats")
		}

		if stats.TotalMessages == 0 {
			t.Error("Should have at least one message in stats")
		}
	})
}