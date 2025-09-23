package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/daemon"
	"github.com/jasperwreed/ai-memory/internal/tui"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewBrowseCommand() *cobra.Command {
	var useAll bool
	var noCapture bool

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse conversations in TUI",
		Long:  `Open an interactive terminal UI to browse and search conversations.`,
		Example: `  # Browse default conversations
  mem browse

  # Browse all imported conversations
  mem browse --all

  # Browse specific database
  mem browse --db custom.db

  # Browse without starting auto-capture daemon
  mem browse --no-capture`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrowse(dbPath, useAll, !noCapture)
		},
	}

	cmd.Flags().BoolVar(&useAll, "all", false, "Browse all imported conversations (all_conversations.db)")
	cmd.Flags().BoolVar(&noCapture, "no-capture", false, "Don't start auto-capture daemon")

	return cmd
}

func runBrowse(customDB string, useAll bool, startCapture bool) error {
	database := customDB

	// If --all flag is used and no custom DB specified, use all_conversations.db
	if useAll && customDB == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		database = filepath.Join(homeDir, ".ai-memory", "all_conversations.db")
	}

	// Start auto-capture daemon if requested
	var captureDaemon *daemon.CaptureDaemon
	if startCapture {
		config := daemon.DefaultConfig()
		d, err := daemon.NewCaptureDaemon(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to start auto-capture daemon: %v\n", err)
		} else {
			if err := d.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to start auto-capture daemon: %v\n", err)
			} else {
				captureDaemon = d
				fmt.Println("Auto-capture daemon started in background")
				defer func() {
					fmt.Println("Stopping auto-capture daemon...")
					if err := captureDaemon.Stop(); err != nil {
						fmt.Fprintf(os.Stderr, "Error stopping daemon: %v\n", err)
					}
				}()
			}
		}
	}

	store, err := storage.NewSQLiteStore(database)
	if err != nil {
		return err
	}
	defer store.Close()

	browser := tui.NewBrowser(store)
	return browser.Run()
}