package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jasperwreed/ai-memory/internal/storage"
)

func TestNewSearcher(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-searcher-*")
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

	searcher := NewSearcher(store)
	if searcher == nil {
		t.Error("NewSearcher() returned nil")
	}
	if searcher.store != store {
		t.Error("NewSearcher() did not set store correctly")
	}
}

func TestSearcher_EmptyDatabase(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-empty-*")
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

	searcher := NewSearcher(store)

	// Search in empty database
	results, err := searcher.Search("anything", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Search() in empty database returned %d results, want 0", len(results))
	}

	// SearchWithFilters in empty database
	filters := map[string]interface{}{"tool": "claude"}
	results, err = searcher.SearchWithFilters("anything", 10, filters)
	if err != nil {
		t.Fatalf("SearchWithFilters() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchWithFilters() in empty database returned %d results, want 0", len(results))
	}
}

func TestSearcher_SearchWithFilters_InvalidTypes(t *testing.T) {
	// Create temp database
	tempDir, err := os.MkdirTemp("", "test-search-invalid-*")
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

	searcher := NewSearcher(store)

	// Test with invalid filter types (non-string values)
	filters := map[string]interface{}{
		"tool":    123,        // Invalid: number instead of string
		"project": []string{}, // Invalid: slice instead of string
	}

	// Should not panic and should handle gracefully
	results, err := searcher.SearchWithFilters("test", 10, filters)
	if err != nil {
		t.Fatalf("SearchWithFilters() error = %v", err)
	}

	// Should return empty results since database is empty
	if len(results) != 0 {
		t.Errorf("SearchWithFilters() with invalid filters should return 0 results from empty db, got %d", len(results))
	}
}