package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/jasper/ai-memory/internal/storage"
)

func NewListCommand() *cobra.Command {
	var limit int
	var filterTool string
	var filterProject string
	var useAll bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent conversations",
		Long:  `List recent AI conversations with filtering options.`,
		Example: `  # List recent conversations
  mem list

  # List all imported conversations
  mem list --all

  # List conversations from a specific tool
  mem list --tool claude

  # List conversations from a specific project
  mem list --project backend --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := make(map[string]string)
			if filterTool != "" {
				filter["tool"] = filterTool
			}
			if filterProject != "" {
				filter["project"] = filterProject
			}
			return runList(limit, filter, dbPath, useAll)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of conversations to list")
	cmd.Flags().StringVar(&filterTool, "tool", "", "Filter by tool")
	cmd.Flags().StringVar(&filterProject, "project", "", "Filter by project")
	cmd.Flags().BoolVar(&useAll, "all", false, "List from all imported conversations (all_conversations.db)")

	return cmd
}

func runList(limit int, filter map[string]string, customDB string, useAll bool) error {
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