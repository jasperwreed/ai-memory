package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/tui"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewBrowseCommand() *cobra.Command {
	var useAll bool

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse conversations in TUI",
		Long:  `Open an interactive terminal UI to browse and search conversations.`,
		Example: `  # Browse default conversations
  mem browse

  # Browse all imported conversations
  mem browse --all

  # Browse specific database
  mem browse --db custom.db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrowse(dbPath, useAll)
		},
	}

	cmd.Flags().BoolVar(&useAll, "all", false, "Browse all imported conversations (all_conversations.db)")

	return cmd
}

func runBrowse(customDB string, useAll bool) error {
	database := customDB

	// If --all flag is used and no custom DB specified, use all_conversations.db
	if useAll && customDB == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		database = filepath.Join(homeDir, ".ai-memory", "all_conversations.db")
	}

	store, err := storage.NewSQLiteStore(database)
	if err != nil {
		return err
	}
	defer store.Close()

	browser := tui.NewBrowser(store)
	return browser.Run()
}