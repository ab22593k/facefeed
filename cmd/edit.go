package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"

	"github.com/spf13/cobra"
)

// editCmd represents the edit command.
var editCmd = &cobra.Command{
	Use:   "edit <post_id> <new_message>",
	Short: "Edit a Facebook post's message",
	Long: `Update the message/content of an existing Facebook post.

Edits are limited to the message text. Not all post types support editing
(e.g. photos and videos attached to a post cannot be changed after publishing).

Examples:
  facefeed edit 123456789_987654321 "Updated message"
  facefeed edit post_id_123 "Fixed a typo in the announcement"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envToken := os.Getenv("FB_ACCESS_TOKEN")
		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		postID := args[0]
		newMessage := args[1]

		theme.PrintSection("Edit Post")
		theme.Info("Post ID", postID)
		theme.Info("New Message", newMessage)

		if err := FBClient.UpdatePost(postID, newMessage); err != nil {
			theme.Error(fmt.Sprintf("Failed to edit post: %v", err))
			os.Exit(1)
		}
		theme.Success("Post updated successfully!")
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
