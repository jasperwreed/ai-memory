package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/capture"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewCaptureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture AI conversation from stdin",
		Long:  `Capture AI conversation from stdin and save it to the database.`,
		Example: `  # Capture from pipe
  claude | ai-memory capture --tool claude --project myapp

  # Capture with tags
  ai-memory capture --tool aider --project backend --tags "auth,debugging" < conversation.txt`,
		RunE: runCapture,
	}

	cmd.Flags().StringVar(&tool, "tool", "", "AI tool name (claude, aider, gpt, etc.)")
	cmd.Flags().StringVar(&project, "project", "", "Project name")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Comma-separated tags")
	cmd.Flags().Bool("auto-detect", false, "Auto-detect tool from input")

	return cmd
}

func runCapture(cmd *cobra.Command, args []string) error {
	autoDetect, _ := cmd.Flags().GetBool("auto-detect")

	if tool == "" && !autoDetect {
		return fmt.Errorf("--tool flag is required unless --auto-detect is used")
	}

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	capturer := capture.NewCapturer(tool, project, tags)
	conversation, err := capturer.CaptureFromReader(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to capture conversation: %w", err)
	}

	if autoDetect && tool == "" {
		detectedTool := capture.DetectToolFromInput(conversation.Messages[0].Content)
		conversation.Tool = detectedTool
		fmt.Fprintf(os.Stderr, "Auto-detected tool: %s\n", detectedTool)
	}

	if err := store.SaveConversation(conversation); err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	fmt.Printf("✓ Captured conversation (ID: %d)\n", conversation.ID)
	fmt.Printf("  Title: %s\n", conversation.Title)
	fmt.Printf("  Tool: %s\n", conversation.Tool)
	if conversation.Project != "" {
		fmt.Printf("  Project: %s\n", conversation.Project)
	}
	if len(conversation.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(conversation.Tags, ", "))
	}
	fmt.Printf("  Messages: %d\n", len(conversation.Messages))

	totalTokens := 0
	for _, msg := range conversation.Messages {
		totalTokens += msg.TokenCount
	}
	fmt.Printf("  Estimated tokens: %d\n", totalTokens)

	return nil
}