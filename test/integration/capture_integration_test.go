//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jasperwreed/ai-memory/internal/models"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func TestCaptureIntegration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp database directory
	tempDir, err := os.MkdirTemp("", "test-capture-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testDbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	store, err := storage.NewSQLiteStore(testDbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Add a test conversation
	conv := &models.Conversation{
		Title:   "Test Conversation",
		Tool:    "claude",
		Project: "test-project",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
	}

	if err := store.SaveConversation(conv); err != nil {
		t.Fatal(err)
	}

	// Verify save worked
	conversations, err := store.ListConversations(1, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(conversations) != 1 {
		t.Errorf("Expected 1 conversation, got %d", len(conversations))
	}

	if conversations[0].Title != "Test Conversation" {
		t.Errorf("Title = %v, want Test Conversation", conversations[0].Title)
	}
}

func TestCaptureWithTagsIntegration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp database directory
	tempDir, err := os.MkdirTemp("", "test-capture-tags-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testDbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	store, err := storage.NewSQLiteStore(testDbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Add a test conversation with tags
	conv := &models.Conversation{
		Title:   "Tagged Conversation",
		Tool:    "gpt",
		Project: "test-project",
		Tags:    []string{"test", "integration", "capture"},
		Messages: []models.Message{
			{Role: "user", Content: "Test message"},
			{Role: "assistant", Content: "Test response"},
		},
	}

	if err := store.SaveConversation(conv); err != nil {
		t.Fatal(err)
	}

	// Retrieve and verify
	conversations, err := store.ListConversations(1, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(conversations) != 1 {
		t.Fatalf("Expected 1 conversation, got %d", len(conversations))
	}

	saved := conversations[0]
	if saved.Title != "Tagged Conversation" {
		t.Errorf("Title = %v, want Tagged Conversation", saved.Title)
	}

	if saved.Tool != "gpt" {
		t.Errorf("Tool = %v, want gpt", saved.Tool)
	}

	if len(saved.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(saved.Tags))
	}
}