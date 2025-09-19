package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/jasper/ai-memory/internal/storage"
)

func NewStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics about captured conversations",
		Long:  `Display statistics about your AI conversations including token usage and costs.`,
		RunE:  runStats,
	}

	return cmd
}

func runStats(cmd *cobra.Command, args []string) error {
	store, err := storage.NewSQLiteStore(dbPath)
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