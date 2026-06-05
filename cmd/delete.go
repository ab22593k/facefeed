package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"

	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command.
var deleteCmd = &cobra.Command{
	Use:   "delete <post_id>",
	Short: "Delete a Facebook post by ID",
	Long: `Delete a specific Facebook post, scheduled post, or draft by its ID.

This is a simpler, more direct alternative to 'rollback' — it takes a single
post ID and deletes it immediately without needing page/group configuration.

The post ID is the one shown by 'scheduled' or 'promotable' commands.

Examples:
  facefeed delete 123456789_987654321
  facefeed delete 987654321_123456789`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envToken := os.Getenv("FB_ACCESS_TOKEN")

		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		postID := args[0]
		if err := FBClient.DeletePostByID(postID); err != nil {
			theme.Error(fmt.Sprintf("Failed to delete post: %v", err))
			os.Exit(1)
		}
		theme.Success(fmt.Sprintf("Successfully deleted post: %s", postID))
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
