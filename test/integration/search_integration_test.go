//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jasperwreed/ai-memory/internal/models"
	"github.com/jasperwreed/ai-memory/internal/search"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func TestSearchIntegration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Add test data
	conversations := []*models.Conversation{
		{
			Title:     "Python Programming",
			Tool:      "claude",
			Project:   "python-app",
			CreatedAt: time.Now(),
			Messages: []models.Message{
				{Role: "user", Content: "How do I use Python decorators?"},
				{Role: "assistant", Content: "Python decorators are functions that modify other functions..."},
			},
		},
		{
			Title:     "Go Concurrency",
			Tool:      "gpt",
			Project:   "go-service",
			CreatedAt: time.Now(),
			Messages: []models.Message{
				{Role: "user", Content: "Explain Go channels and goroutines"},
				{Role: "assistant", Content: "Go channels are used for communication between goroutines..."},
			},
		},
		{
			Title:     "JavaScript Async",
			Tool:      "claude",
			Project:   "web-app",
			CreatedAt: time.Now(),
			Messages: []models.Message{
				{Role: "user", Content: "What's the difference between async/await and promises?"},
				{Role: "assistant", Content: "Async/await is syntactic sugar over promises..."},
			},
		},
	}

	for _, conv := range conversations {
		if err := store.SaveConversation(conv); err != nil {
			t.Fatal(err)
		}
	}

	searcher := search.NewSearcher(store)

	t.Run("search with results", func(t *testing.T) {
		results, err := searcher.Search("Python", 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected at least one result for 'Python' search")
		}

		// Check that Python Programming is in results
		found := false
		for _, result := range results {
			if result.Conversation.Title == "Python Programming" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'Python Programming' in search results")
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		results, err := searcher.Search("Rust ownership", 10)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected no results for 'Rust ownership', got %d", len(results))
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		results, err := searcher.Search("programming OR concurrency OR async", 1)
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		if len(results) > 1 {
			t.Errorf("Expected at most 1 result with limit=1, got %d", len(results))
		}
	})
}

func TestSearchWithFiltersIntegration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-filters-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Add test data with different tools and projects
	conversations := []*models.Conversation{
		{
			Title:     "Claude Test 1",
			Tool:      "claude",
			Project:   "project-a",
			CreatedAt: time.Now(),
			Messages: []models.Message{
				{Role: "user", Content: "Test message about testing"},
			},
		},
		{
			Title:     "Claude Test 2",
			Tool:      "claude",
			Project:   "project-b",
			CreatedAt: time.Now(),
			Messages: []models.Message{
				{Role: "user", Content: "Another test message"},
			},
		},
		{
			Title:     "GPT Test 1",
			Tool:      "gpt",
			Project:   "project-a",
			CreatedAt: time.Now(),
			Messages: []models.Message{
				{Role: "user", Content: "Testing with GPT"},
			},
		},
	}

	for _, conv := range conversations {
		if err := store.SaveConversation(conv); err != nil {
			t.Fatal(err)
		}
	}

	searcher := search.NewSearcher(store)

	t.Run("filter by tool", func(t *testing.T) {
		results, err := searcher.SearchWithFilters("test", 10, map[string]interface{}{
			"tool": "claude",
		})
		if err != nil {
			t.Fatalf("SearchWithFilters() error = %v", err)
		}

		// All results should be from claude
		for _, result := range results {
			if result.Conversation.Tool != "claude" {
				t.Errorf("Expected tool 'claude', got %s", result.Conversation.Tool)
			}
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		results, err := searcher.SearchWithFilters("test", 10, map[string]interface{}{
			"project": "project-a",
		})
		if err != nil {
			t.Fatalf("SearchWithFilters() error = %v", err)
		}

		// All results should be from project-a
		for _, result := range results {
			if result.Conversation.Project != "project-a" {
				t.Errorf("Expected project 'project-a', got %s", result.Conversation.Project)
			}
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		results, err := searcher.SearchWithFilters("test", 10, map[string]interface{}{
			"tool":    "claude",
			"project": "project-b",
		})
		if err != nil {
			t.Fatalf("SearchWithFilters() error = %v", err)
		}

		// Should find exactly the Claude Test 2
		if len(results) != 1 {
			t.Errorf("Expected 1 result with combined filters, got %d", len(results))
		}
		if len(results) > 0 && results[0].Conversation.Title != "Claude Test 2" {
			t.Errorf("Expected 'Claude Test 2', got %s", results[0].Conversation.Title)
		}
	})
}