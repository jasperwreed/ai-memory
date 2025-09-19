package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/search"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewSearchCommand() *cobra.Command {
	var limit int
	var showContext bool
	var useAll bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search conversations",
		Long:  `Search through all captured AI conversations using full-text search.`,
		Example: `  # Search for authentication-related conversations
  mem search "authentication JWT"

  # Search in all imported conversations
  mem search "database migration" --all

  # Search with limited results
  mem search "database migration" --limit 5

  # Search with full context
  mem search "error handling" --context`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runSearch(query, limit, showContext, dbPath, useAll)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of results")
	cmd.Flags().BoolVar(&showContext, "context", false, "Show full message context")
	cmd.Flags().BoolVar(&useAll, "all", false, "Search in all imported conversations (all_conversations.db)")

	return cmd
}

func runSearch(query string, limit int, showContext bool, customDB string, useAll bool) error {
	database := customDB

	// If --all flag is used and no custom DB specified, use all_conversations.db
	if useAll && customDB == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		database = filepath.Join(homeDir, ".ai-memory", "all_conversations.db")
	}

	store, err := storage.NewSQLiteStore(database)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	searcher := search.NewSearcher(store)
	results, err := searcher.Search(query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	fmt.Printf("Found %d result(s) for '%s':\n\n", len(results), query)

	for i, result := range results {
		fmt.Printf("%d. [ID: %d] %s\n", i+1, result.Conversation.ID, result.Conversation.Title)
		fmt.Printf("   Tool: %s", result.Conversation.Tool)
		if result.Conversation.Project != "" {
			fmt.Printf(" | Project: %s", result.Conversation.Project)
		}
		fmt.Printf(" | %s\n", result.Conversation.CreatedAt.Format("2006-01-02 15:04"))

		if showContext {
			fmt.Printf("\n   %s\n", strings.ReplaceAll(result.Snippet, "\n", "\n   "))
		} else {
			snippet := result.Snippet
			if len(snippet) > 100 {
				snippet = snippet[:100] + "..."
			}
			fmt.Printf("   %s\n", snippet)
		}
		fmt.Println()
	}

	return nil
}