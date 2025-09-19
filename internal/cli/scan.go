package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/jasper/ai-memory/internal/scanner"
	"github.com/jasper/ai-memory/internal/storage"
)

func NewScanCommand() *cobra.Command {
	var outputDB string
	var verbose bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan system for AI tool conversation files",
		Long: `Automatically discover and import conversations from AI CLI tools installed on your system.
Currently supports: Claude Code

The scan command will:
1. Search for Claude Code session files in ~/.claude/projects/
2. Parse and import all found conversations
3. Create or update the conversations database`,
		Example: `  # Scan and import all found conversations
  mem scan

  # Scan with custom database location
  mem scan --output ~/my-conversations.db

  # Dry run to see what would be imported
  mem scan --dry-run

  # Verbose output
  mem scan --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(outputDB, verbose, dryRun)
		},
	}

	cmd.Flags().StringVar(&outputDB, "output", "", "Output database file (default: ~/.ai-memory/all_conversations.db)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed progress")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be imported without actually importing")

	return cmd
}

func runScan(outputDB string, verbose, dryRun bool) error {
	if outputDB == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		outputDB = filepath.Join(homeDir, ".ai-memory", "all_conversations.db")
	}

	fmt.Println("🔍 Scanning for AI conversation files...")
	fmt.Println()

	// Initialize scanners (extensible design for future tools)
	scanners := []scanner.Scanner{
		scanner.NewClaudeScanner(),
		// Future: scanner.NewAiderScanner(),
		// Future: scanner.NewCursorScanner(),
		// Future: scanner.NewGPTCLIScanner(),
	}

	totalFound := 0
	totalImported := 0
	var results []scanner.ScanResult

	// Scan each tool
	for _, s := range scanners {
		if verbose {
			fmt.Printf("Scanning %s...\n", s.Name())
		}

		sessions, err := s.ScanForSessions()
		if err != nil {
			if verbose {
				fmt.Printf("  ⚠️  Error scanning %s: %v\n", s.Name(), err)
			}
			continue
		}

		if len(sessions) == 0 {
			if verbose {
				fmt.Printf("  No %s sessions found\n", s.Name())
			}
			continue
		}

		fmt.Printf("📁 Found %d %s session(s)\n", len(sessions), s.Name())
		totalFound += len(sessions)

		if dryRun {
			for _, session := range sessions {
				fmt.Printf("  • %s\n", session.Path)
				if verbose {
					fmt.Printf("    Project: %s, Size: %d bytes, Modified: %s\n",
						session.ProjectName, session.Size, session.ModTime)
				}
			}
			continue
		}

		// Import sessions
		result := scanner.ScanResult{
			Tool:          s.Name(),
			SessionsFound: len(sessions),
		}

		if !dryRun {
			imported, failed := importSessions(s, sessions, outputDB, verbose)
			result.Imported = imported
			result.Failed = failed
			totalImported += imported
		}

		results = append(results, result)
	}

	// Summary
	fmt.Println()
	fmt.Println("═══════════════════════════════════")
	fmt.Printf("📊 Scan Complete\n")
	fmt.Printf("   Total sessions found: %d\n", totalFound)

	if !dryRun {
		fmt.Printf("   Successfully imported: %d\n", totalImported)
		if totalFound > totalImported {
			fmt.Printf("   Failed to import: %d\n", totalFound-totalImported)
		}
		fmt.Printf("   Database: %s\n", outputDB)
		fmt.Println()
		fmt.Println("✨ You can now use:")
		fmt.Println("   mem browse        # Browse all imported conversations")
		fmt.Println("   mem search <query> # Search across all conversations")
		fmt.Println("   mem stats         # View statistics")
	} else {
		fmt.Println("\n(Dry run - no changes made)")
		fmt.Printf("Would import %d conversations to: %s\n", totalFound, outputDB)
	}

	return nil
}

func importSessions(s scanner.Scanner, sessions []scanner.SessionInfo, dbPath string, verbose bool) (imported, failed int) {
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		fmt.Printf("  ❌ Failed to open database: %v\n", err)
		return 0, len(sessions)
	}
	defer store.Close()

	for i, session := range sessions {
		if verbose {
			fmt.Printf("  [%d/%d] Importing %s...\n", i+1, len(sessions), filepath.Base(session.Path))
		}

		conv, err := s.ParseSession(session.Path)
		if err != nil {
			if verbose {
				fmt.Printf("    ⚠️  Failed to parse: %v\n", err)
			}
			failed++
			continue
		}

		// Check if already imported (avoid duplicates)
		existing, _ := store.ListConversations(100, 0, map[string]string{
			"tool": conv.Tool,
		})

		isDuplicate := false
		for _, e := range existing {
			if e.Title == conv.Title && len(e.Messages) == len(conv.Messages) {
				isDuplicate = true
				break
			}
		}

		if isDuplicate {
			if verbose {
				fmt.Printf("    ⏭️  Skipping duplicate\n")
			}
			continue
		}

		if err := store.SaveConversation(conv); err != nil {
			if verbose {
				fmt.Printf("    ❌ Failed to save: %v\n", err)
			}
			failed++
			continue
		}

		imported++
		if !verbose && imported%10 == 0 {
			fmt.Printf("  Imported %d/%d...\n", imported, len(sessions))
		}
	}

	return imported, failed
}