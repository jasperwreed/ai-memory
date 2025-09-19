package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/jasper/ai-memory/internal/storage"
)

func NewListCommand() *cobra.Command {
	var limit int
	var filterTool string
	var filterProject string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent conversations",
		Long:  `List recent AI conversations with filtering options.`,
		Example: `  # List recent conversations
  ai-memory list

  # List conversations from a specific tool
  ai-memory list --tool claude

  # List conversations from a specific project
  ai-memory list --project backend --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := make(map[string]string)
			if filterTool != "" {
				filter["tool"] = filterTool
			}
			if filterProject != "" {
				filter["project"] = filterProject
			}
			return runList(limit, filter)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of conversations to list")
	cmd.Flags().StringVar(&filterTool, "tool", "", "Filter by tool")
	cmd.Flags().StringVar(&filterProject, "project", "", "Filter by project")

	return cmd
}

func runList(limit int, filter map[string]string) error {
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	conversations, err := store.ListConversations(limit, 0, filter)
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	if len(conversations) == 0 {
		fmt.Println("No conversations found.")
		return nil
	}

	fmt.Printf("Recent conversations:\n\n")

	for _, conv := range conversations {
		fmt.Printf("[ID: %d] %s\n", conv.ID, conv.Title)
		fmt.Printf("  Tool: %s", conv.Tool)
		if conv.Project != "" {
			fmt.Printf(" | Project: %s", conv.Project)
		}
		if len(conv.Tags) > 0 {
			fmt.Printf(" | Tags: %s", strings.Join(conv.Tags, ", "))
		}
		fmt.Printf("\n  Created: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}