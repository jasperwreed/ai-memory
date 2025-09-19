package search

import (
	"github.com/jasperwreed/ai-memory/internal/models"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

type Searcher struct {
	store *storage.SQLiteStore
}

func NewSearcher(store *storage.SQLiteStore) *Searcher {
	return &Searcher{store: store}
}

func (s *Searcher) Search(query string, limit int) ([]models.SearchResult, error) {
	return s.store.Search(query, limit)
}

func (s *Searcher) SearchWithFilters(query string, limit int, filters map[string]interface{}) ([]models.SearchResult, error) {
	results, err := s.store.Search(query, limit)
	if err != nil {
		return nil, err
	}

	if tool, ok := filters["tool"].(string); ok && tool != "" {
		filtered := []models.SearchResult{}
		for _, r := range results {
			if r.Conversation.Tool == tool {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if project, ok := filters["project"].(string); ok && project != "" {
		filtered := []models.SearchResult{}
		for _, r := range results {
			if r.Conversation.Project == project {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	return results, nil
}