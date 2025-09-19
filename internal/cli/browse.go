package cli

import (
	"github.com/spf13/cobra"
	"github.com/jasper/ai-memory/internal/tui"
	"github.com/jasper/ai-memory/internal/storage"
)

func NewBrowseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse conversations in TUI",
		Long:  `Open an interactive terminal UI to browse and search conversations.`,
		RunE:  runBrowse,
	}

	return cmd
}

func runBrowse(cmd *cobra.Command, args []string) error {
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	browser := tui.NewBrowser(store)
	return browser.Run()
}