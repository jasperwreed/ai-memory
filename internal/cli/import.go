package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/jasper/ai-memory/internal/capture"
	"github.com/jasper/ai-memory/internal/storage"
)

func NewImportCommand() *cobra.Command {
	var sessionFile string
	var claudeCodeProject string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import Claude Code session files",
		Long:  `Import conversations from Claude Code session files (JSONL format).`,
		Example: `  # Import a specific Claude Code session file
  ai-memory import --file ~/.claude/projects/myproject/session.jsonl

  # Import from current project's Claude Code session
  ai-memory import --claude-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sessionFile == "" && !cmd.Flags().Changed("claude-project") {
				return fmt.Errorf("either --file or --claude-project flag is required")
			}

			if cmd.Flags().Changed("claude-project") {
				return importClaudeProject(claudeCodeProject)
			}

			return importSessionFile(sessionFile)
		},
	}

	cmd.Flags().StringVar(&sessionFile, "file", "", "Path to Claude Code session file (JSONL)")
	cmd.Flags().StringVar(&claudeCodeProject, "claude-project", "", "Import from Claude project directory")

	return cmd
}

func importSessionFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	parser := capture.NewClaudeCodeParser()
	conversation, err := parser.ParseJSONL(file)
	if err != nil {
		return fmt.Errorf("failed to parse session file: %w", err)
	}

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	if err := store.SaveConversation(conversation); err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	fmt.Printf("✓ Imported Claude Code session (ID: %d)\n", conversation.ID)
	fmt.Printf("  Title: %s\n", conversation.Title)
	fmt.Printf("  Project: %s\n", conversation.Project)
	fmt.Printf("  Messages: %d\n", len(conversation.Messages))

	return nil
}

func importClaudeProject(projectName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeDir := filepath.Join(homeDir, ".claude", "projects")

	if projectName == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName = strings.ReplaceAll(cwd, "/", "-")
	}

	projectDir := filepath.Join(claudeDir, projectName)

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return fmt.Errorf("failed to read Claude project directory: %w", err)
	}

	imported := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			filePath := filepath.Join(projectDir, entry.Name())
			fmt.Printf("Importing %s...\n", entry.Name())
			if err := importSessionFile(filePath); err != nil {
				fmt.Printf("  Warning: %v\n", err)
			} else {
				imported++
			}
		}
	}

	if imported == 0 {
		return fmt.Errorf("no Claude Code sessions found in %s", projectDir)
	}

	fmt.Printf("\n✓ Successfully imported %d session(s)\n", imported)
	return nil
}