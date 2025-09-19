package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

func NewDeleteCommand() *cobra.Command {
	var conversationID int64
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a conversation",
		Long:  `Delete a conversation from the database.`,
		Example: `  # Delete a conversation with confirmation
  ai-memory delete --id 42

  # Delete without confirmation prompt
  ai-memory delete --id 42 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if conversationID == 0 {
				return fmt.Errorf("--id flag is required")
			}
			return runDelete(conversationID, confirm)
		},
	}

	cmd.Flags().Int64Var(&conversationID, "id", 0, "Conversation ID to delete")
	cmd.Flags().BoolVar(&confirm, "yes", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("id")

	return cmd
}

func runDelete(id int64, skipConfirm bool) error {
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	conversation, err := store.GetConversation(id)
	if err != nil {
		return fmt.Errorf("conversation not found: %w", err)
	}

	if !skipConfirm {
		fmt.Printf("Delete conversation '%s' (ID: %d)? [y/N]: ", conversation.Title, id)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := store.DeleteConversation(id); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	fmt.Printf("✓ Deleted conversation (ID: %d)\n", id)
	return nil
}