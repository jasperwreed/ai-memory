package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasperwreed/ai-memory/internal/models"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func TestNewSearchCommand(t *testing.T) {
	cmd := NewSearchCommand()

	if cmd.Use != "search <query>" {
		t.Errorf("Command.Use = %v, want %v", cmd.Use, "search <query>")
	}

	if cmd.Short == "" {
		t.Error("Command.Short should not be empty")
	}

	// Check that required flags are defined
	flags := []string{"limit", "context", "all"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Flag %q not defined", flag)
		}
	}

	// Check default values
	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag.DefValue != "10" {
		t.Errorf("Default limit = %v, want 10", limitFlag.DefValue)
	}
}

func TestRunSearch_NoResults(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run search
	err = runSearch("nonexistent query", 10, false, dbPath, false)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("runSearch() error = %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No results found") {
		t.Errorf("Expected 'No results found' in output, got: %s", output)
	}
}

func TestRunSearch_WithResults(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-with-results-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database and add test data
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	// Add test conversations
	conv1 := &models.Conversation{
		Title:     "Test Authentication Discussion",
		Tool:      "claude",
		Project:   "backend",
		CreatedAt: time.Now(),
		Messages: []models.Message{
			{
				Role:    "user",
				Content: "How do I implement JWT authentication?",
			},
			{
				Role:    "assistant",
				Content: "To implement JWT authentication, you need to...",
			},
		},
	}

	conv2 := &models.Conversation{
		Title:     "Database Migration Help",
		Tool:      "gpt",
		Project:   "database",
		CreatedAt: time.Now(),
		Messages: []models.Message{
			{
				Role:    "user",
				Content: "I need help with database migration scripts",
			},
			{
				Role:    "assistant",
				Content: "Database migration involves...",
			},
		},
	}

	if err := store.SaveConversation(conv1); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConversation(conv2); err != nil {
		t.Fatal(err)
	}
	store.Close()

	// Test search with results
	tests := []struct {
		name        string
		query       string
		limit       int
		showContext bool
		expectFound bool
		expectCount int
	}{
		{
			name:        "search for authentication",
			query:       "authentication",
			limit:       10,
			showContext: false,
			expectFound: true,
			expectCount: 1,
		},
		{
			name:        "search for database",
			query:       "database",
			limit:       10,
			showContext: false,
			expectFound: true,
			expectCount: 1,
		},
		{
			name:        "search with limit",
			query:       "help",
			limit:       1,
			showContext: false,
			expectFound: true,
			expectCount: 1,
		},
		{
			name:        "search with context",
			query:       "JWT",
			limit:       10,
			showContext: true,
			expectFound: true,
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run search
			err = runSearch(tt.query, tt.limit, tt.showContext, dbPath, false)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Errorf("runSearch() error = %v", err)
			}

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expectFound {
				if !strings.Contains(output, "Found") {
					t.Errorf("Expected results to be found for query %q, got: %s", tt.query, output)
				}
			}
		})
	}
}

func TestRunSearch_DatabasePath(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test-search-db-path-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		customDB  string
		useAll    bool
		expectErr bool
	}{
		{
			name:      "custom database path",
			customDB:  filepath.Join(tempDir, "custom.db"),
			useAll:    false,
			expectErr: false,
		},
		{
			name:      "use all conversations db",
			customDB:  "",
			useAll:    true,
			expectErr: false,
		},
		{
			name:      "invalid database path",
			customDB:  "/invalid/path/that/does/not/exist.db",
			useAll:    false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create custom DB if needed
			if tt.customDB != "" && !tt.expectErr {
				store, err := storage.NewSQLiteStore(tt.customDB)
				if err != nil {
					t.Fatal(err)
				}
				store.Close()
			}

			err := runSearch("test query", 10, false, tt.customDB, tt.useAll)

			if (err != nil) != tt.expectErr {
				t.Errorf("runSearch() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRunSearch_OutputFormatting(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-format-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database and add test data
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	// Add test conversation with long content
	longContent := strings.Repeat("This is a very long message content. ", 20)
	conv := &models.Conversation{
		Title:     "Long Content Test",
		Tool:      "claude",
		Project:   "test-project",
		CreatedAt: time.Now(),
		Tags:      []string{"tag1", "tag2"},
		Messages: []models.Message{
			{
				Role:    "user",
				Content: longContent,
			},
		},
	}

	if err := store.SaveConversation(conv); err != nil {
		t.Fatal(err)
	}
	store.Close()

	// Test without context (should truncate)
	t.Run("truncated snippet", func(t *testing.T) {
		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err = runSearch("long message", 10, false, dbPath, false)

		w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Errorf("runSearch() error = %v", err)
		}

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should contain truncation indicator
		if !strings.Contains(output, "...") {
			t.Error("Expected truncated output to contain '...'")
		}

		// Should show project
		if !strings.Contains(output, "test-project") {
			t.Error("Expected output to contain project name")
		}

		// Should show tool
		if !strings.Contains(output, "claude") {
			t.Error("Expected output to contain tool name")
		}
	})

	// Test with context (should not truncate)
	t.Run("full context", func(t *testing.T) {
		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err = runSearch("long message", 10, true, dbPath, false)

		w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Errorf("runSearch() error = %v", err)
		}

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should contain more content when context is shown
		if len(output) < 200 {
			t.Error("Expected longer output when context is enabled")
		}
	})
}

