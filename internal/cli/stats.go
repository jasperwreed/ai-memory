package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewStatsCommand() *cobra.Command {
	var useAll bool

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics about captured conversations",
		Long:  `Display statistics about your AI conversations including token usage and costs.`,
		Example: `  # Show stats for default database
  mem stats

  # Show stats for all imported conversations
  mem stats --all

  # Show stats for specific database
  mem stats --db custom.db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStats(dbPath, useAll)
		},
	}

	cmd.Flags().BoolVar(&useAll, "all", false, "Show stats for all imported conversations (all_conversations.db)")

	return cmd
}

func runStats(customDB string, useAll bool) error {
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

	stats, err := store.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	fmt.Println("AI Memory Statistics")
	fmt.Println("====================")
	fmt.Printf("\nTotal Conversations: %d\n", stats.TotalConversations)
	fmt.Printf("Total Messages: %d\n", stats.TotalMessages)
	fmt.Printf("Total Tokens: %d\n", stats.TotalTokens)
	fmt.Printf("Estimated Cost: $%.4f\n", stats.EstimatedCost)

	if len(stats.ToolBreakdown) > 0 {
		fmt.Println("\nConversations by Tool:")
		for tool, count := range stats.ToolBreakdown {
			fmt.Printf("  %s: %d\n", tool, count)
		}
	}

	if len(stats.ProjectBreakdown) > 0 {
		fmt.Println("\nConversations by Project:")
		for project, count := range stats.ProjectBreakdown {
			fmt.Printf("  %s: %d\n", project, count)
		}
	}

	return nil
}