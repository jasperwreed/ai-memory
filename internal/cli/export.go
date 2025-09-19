package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewExportCommand() *cobra.Command {
	var conversationID int64
	var format string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export conversation as JSON",
		Long:  `Export a conversation in JSON format for sharing or backup.`,
		Example: `  # Export a specific conversation
  ai-memory export --id 42

  # Export to file
  ai-memory export --id 42 > conversation.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if conversationID == 0 {
				return fmt.Errorf("--id flag is required")
			}
			return runExport(conversationID, format)
		},
	}

	cmd.Flags().Int64Var(&conversationID, "id", 0, "Conversation ID to export")
	cmd.Flags().StringVar(&format, "format", "json", "Export format (currently only json)")
	cmd.MarkFlagRequired("id")

	return cmd
}

func runExport(id int64, format string) error {
	if format != "json" {
		return fmt.Errorf("only JSON format is currently supported")
	}

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	conversation, err := store.GetConversation(id)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	output, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	fmt.Println(string(output))
	return nil
}