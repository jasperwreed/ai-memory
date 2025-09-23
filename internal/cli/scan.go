package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/audit"
	"github.com/jasperwreed/ai-memory/internal/scanner"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewScanCommand() *cobra.Command {
	var outputDB string
	var verbose bool
	var dryRun bool
	var auditOnly bool
	var noAudit bool
	var auditDir string

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan system for AI tool conversation files",
		Long: `Automatically discover and import conversations from AI CLI tools installed on your system.
Currently supports: Claude Code

The scan command will:
1. Search for Claude Code session files in ~/.claude/projects/
2. Capture raw sessions to audit logs for permanent preservation (default)
3. Import parsed conversations to the database (default)
4. Link database entries to audit shards for traceability

By default, scan performs BOTH audit capture and database import to protect against
Claude's 30-day purge. Use flags to modify this behavior.`,
		Example: `  # Scan and import all found conversations
  mem scan

  # Scan with custom database location
  mem scan --output ~/my-conversations.db

  # Dry run to see what would be imported
  mem scan --dry-run

  # Verbose output
  mem scan --verbose

  # Only capture to audit logs (no database import)
  mem scan --audit-only

  # Only import to database (no audit capture)
  mem scan --no-audit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate flag combinations
			if auditOnly && noAudit {
				return fmt.Errorf("cannot use --audit-only and --no-audit together")
			}

			// Determine what to run
			captureAudit := !noAudit        // Capture audit by default, unless --no-audit
			importToDB := !auditOnly         // Import to DB by default, unless --audit-only

			return runScan(outputDB, auditDir, verbose, dryRun, captureAudit, importToDB)
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultAuditDir := filepath.Join(homeDir, ".ai-memory", "audit")

	cmd.Flags().StringVar(&outputDB, "output", "", "Output database file (default: ~/.ai-memory/all_conversations.db)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed progress")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be imported without actually importing")
	cmd.Flags().BoolVar(&auditOnly, "audit-only", false, "Only capture to audit logs (no database import)")
	cmd.Flags().BoolVar(&noAudit, "no-audit", false, "Skip audit capture (only import to database)")
	cmd.Flags().StringVar(&auditDir, "audit-dir", defaultAuditDir, "Directory for audit logs")

	return cmd
}

func runScan(outputDB, auditDir string, verbose, dryRun, captureAudit, importToDB bool) error {
	if outputDB == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		outputDB = filepath.Join(homeDir, ".ai-memory", "all_conversations.db")
	}

	fmt.Println("🔍 Scanning for AI conversation files...")

	// Show what will be done
	if dryRun {
		fmt.Println("   Mode: DRY RUN (no changes will be made)")
	} else {
		if captureAudit && importToDB {
			fmt.Println("   Mode: Full capture (audit + database)")
		} else if captureAudit {
			fmt.Println("   Mode: Audit capture only")
		} else if importToDB {
			fmt.Println("   Mode: Database import only")
		}
	}
	fmt.Println()

	// Initialize audit logger if requested
	var auditLogger *audit.AuditLogger
	if captureAudit && !dryRun {
		var err error
		auditLogger, err = audit.NewAuditLogger(auditDir, 100*1024*1024, true) // 100MB shards, compressed
		if err != nil {
			return fmt.Errorf("failed to create audit logger: %w", err)
		}
		defer auditLogger.Close()

		fmt.Printf("📝 Audit logs: %s\n", auditDir)
	}

	// Show database path if importing
	if importToDB && !dryRun {
		fmt.Printf("💾 Database: %s\n", outputDB)
	}

	if (captureAudit || importToDB) && !dryRun {
		fmt.Println()
	}

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
			if importToDB {
				imported, failed := importSessions(s, sessions, outputDB, auditLogger, verbose)
				result.Imported = imported
				result.Failed = failed
				totalImported += imported
			} else if captureAudit {
				// Audit-only mode: just capture raw files
				captureSessionsToAudit(s, sessions, auditLogger, verbose)
				result.Imported = len(sessions)
				totalImported += len(sessions)
			}
		}

		results = append(results, result)
	}

	// Summary
	fmt.Println()
	fmt.Println("═══════════════════════════════════")
	fmt.Printf("📊 Scan Complete\n")
	fmt.Printf("   Total sessions found: %d\n", totalFound)

	if !dryRun {
		if importToDB {
			fmt.Printf("   Successfully imported to DB: %d\n", totalImported)
			if totalFound > totalImported {
				fmt.Printf("   Failed to import: %d\n", totalFound-totalImported)
			}
		}
		if captureAudit {
			fmt.Printf("   Captured to audit logs: %d\n", totalImported)
			fmt.Printf("   Audit directory: %s\n", auditDir)
		}
		if importToDB {
			fmt.Printf("   Database: %s\n", outputDB)
			fmt.Println()
			fmt.Println("✨ You can now use:")
			fmt.Println("   mem browse        # Browse all imported conversations")
			fmt.Println("   mem search <query> # Search across all conversations")
			fmt.Println("   mem stats         # View statistics")
		}
		if captureAudit && !importToDB {
			fmt.Println()
			fmt.Println("✨ Audit logs captured. You can:")
			fmt.Println("   mem daemon logs   # View audit logs")
			fmt.Println("   mem scan --no-audit # Import to database later")
		}
	} else {
		fmt.Println("\n(Dry run - no changes made)")
		actions := []string{}
		if captureAudit {
			actions = append(actions, fmt.Sprintf("capture to %s", auditDir))
		}
		if importToDB {
			actions = append(actions, fmt.Sprintf("import to %s", outputDB))
		}
		if len(actions) > 0 {
			fmt.Printf("Would %s (%d sessions)\n", actions[0], totalFound)
			for i := 1; i < len(actions); i++ {
				fmt.Printf("  and %s\n", actions[i])
			}
		}
	}

	return nil
}

func importSessions(s scanner.Scanner, sessions []scanner.SessionInfo, dbPath string, auditLogger *audit.AuditLogger, verbose bool) (imported, failed int) {
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

		// Capture to audit log if enabled
		if auditLogger != nil {
			// Read the raw file for audit
			rawData, err := os.ReadFile(session.Path)
			if err == nil {
				// Write raw JSONL lines to audit
				lines := splitJSONL(rawData)
				currentShard := getCurrentShardName(auditLogger)

				for _, line := range lines {
					if len(line) > 0 {
						// Add metadata about source
						event := map[string]interface{}{
							"type":        "import",
							"source_path": session.Path,
							"session_id":  conv.SessionID,
							"tool":        conv.Tool,
							"project":     conv.Project,
							"timestamp":   time.Now().Unix(),
						}

						// Store raw line
						auditLogger.WriteRawLine(line)
						auditLogger.WriteEvent(event)
					}
				}

				// Update conversation with audit shard reference
				conv.AuditShard = currentShard
			} else if verbose {
				fmt.Printf("    ⚠️  Could not read file for audit: %v\n", err)
			}
		}

		// Check if already imported (avoid duplicates based on session_id)
		isDuplicate := false
		if conv.SessionID != "" {
			// First check by session_id for exact match
			existing, _ := store.GetConversationBySessionID(conv.SessionID)
			if existing != nil {
				isDuplicate = true
			}
		}

		// Fallback to checking by title and message count
		if !isDuplicate {
			existing, _ := store.ListConversations(100, 0, map[string]string{
				"tool": conv.Tool,
			})
			for _, e := range existing {
				if e.Title == conv.Title && len(e.Messages) == len(conv.Messages) {
					isDuplicate = true
					break
				}
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

// splitJSONL splits raw data into individual JSONL lines
func splitJSONL(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// getCurrentShardName gets the current active shard name from the audit logger
func getCurrentShardName(logger *audit.AuditLogger) string {
	shards, err := logger.GetActiveShards()
	if err != nil || len(shards) == 0 {
		return ""
	}
	// Return the most recent shard
	return filepath.Base(shards[len(shards)-1].Path)
}

// captureSessionsToAudit captures sessions to audit log only (no DB import)
func captureSessionsToAudit(s scanner.Scanner, sessions []scanner.SessionInfo, auditLogger *audit.AuditLogger, verbose bool) {
	for i, session := range sessions {
		if verbose {
			fmt.Printf("  [%d/%d] Capturing %s to audit...\n", i+1, len(sessions), filepath.Base(session.Path))
		}

		// Read the raw file for audit
		rawData, err := os.ReadFile(session.Path)
		if err != nil {
			if verbose {
				fmt.Printf("    ⚠️  Could not read file: %v\n", err)
			}
			continue
		}

		// Parse just to get metadata
		conv, _ := s.ParseSession(session.Path)
		sessionID := ""
		tool := s.Name()
		project := session.ProjectName

		if conv != nil {
			sessionID = conv.SessionID
			tool = conv.Tool
			project = conv.Project
		}

		// Write raw JSONL lines to audit
		lines := splitJSONL(rawData)
		for _, line := range lines {
			if len(line) > 0 {
				// Store raw line
				auditLogger.WriteRawLine(line)

				// Add metadata event
				event := map[string]interface{}{
					"type":        "capture",
					"source_path": session.Path,
					"session_id":  sessionID,
					"tool":        tool,
					"project":     project,
					"timestamp":   time.Now().Unix(),
				}
				auditLogger.WriteEvent(event)
			}
		}

		if verbose {
			fmt.Printf("    ✅ Captured to audit\n")
		}
	}
}