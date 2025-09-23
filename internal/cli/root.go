package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/storage"
	"github.com/jasperwreed/ai-memory/internal/tui"
)

var (
	dbPath  string
	tool    string
	project string
	tags    []string
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "mem [project_dir]",
		Short: "Terminal-based AI conversation manager with vim-like interface",
		Long: `Mem - Capture, search, and manage AI conversations across all CLI-based AI tools.
Never lose valuable solutions or insights from your AI sessions again.

Usage:
  mem           Launch TUI with all conversations (all_conversations.db)
  mem .         Launch TUI with current project's database
  mem <dir>     Launch TUI with specific project's database

Inside the TUI, use vim-style commands:
  :scan         Scan for AI tool conversations
  :search       Search conversations
  :capture      Capture current conversation
  :stats        Show statistics
  :help         Show help`,
		Version: "0.1.0",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runTUI,
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to database file (overrides default behavior)")

	rootCmd.AddCommand(
		NewCaptureCommand(),
		NewSearchCommand(),
		NewListCommand(),
		NewExportCommand(),
		NewBrowseCommand(),
		NewStatsCommand(),
		NewDeleteCommand(),
		NewImportCommand(),
		NewScanCommand(),
		NewDaemonCommand(),
	)

	return rootCmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	validator := NewValidator()
	var database string
	var err error

	if dbPath != "" {
		database = dbPath
	} else if len(args) == 1 {
		database, err = validator.GetProjectDatabasePath(args[0])
		if err != nil {
			return err
		}
	} else {
		database, err = validator.GetDefaultDatabasePath()
		if err != nil {
			return err
		}
	}

	store, err := storage.NewSQLiteStore(database)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	browser := tui.NewBrowserWithPath(store, database)
	return browser.Run()
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}