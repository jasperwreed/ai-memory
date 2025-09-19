package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath  string
	tool    string
	project string
	tags    []string
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ai-memory",
		Short: "Terminal-based AI conversation manager",
		Long: `AI Memory - Capture, search, and manage AI conversations across all CLI-based AI tools.
Never lose valuable solutions or insights from your AI sessions again.`,
		Version: "0.1.0",
	}

	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to database file (default: ~/.ai-memory/conversations.db)")

	rootCmd.AddCommand(
		NewCaptureCommand(),
		NewSearchCommand(),
		NewListCommand(),
		NewExportCommand(),
		NewBrowseCommand(),
		NewStatsCommand(),
		NewDeleteCommand(),
		NewImportCommand(),
	)

	return rootCmd
}

func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}