package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"

	"github.com/spf13/cobra"
)

// replyCmd represents the reply command.
var replyCmd = &cobra.Command{
	Use:   "reply <comment_id> <message>",
	Short: "Reply to a comment on a Facebook post",
	Long: `Post a reply to an existing comment on a Facebook post or page.

The comment ID can be found in the Facebook API response or Graph API Explorer.
Requires pages_manage_posts permission on the access token.

Examples:
  facefeed reply 123456789_987654321 "Thanks for your feedback!"
  facefeed reply comment_id_123 "Happy to help!"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envToken := os.Getenv("FB_ACCESS_TOKEN")
		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		commentID := args[0]
		message := args[1]

		theme.PrintSection("Reply to Comment")
		theme.Info("Comment ID", commentID)

		replyID, err := FBClient.ReplyToComment(commentID, message)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to reply: %v", err))
			os.Exit(1)
		}
		theme.Success(fmt.Sprintf("Replied! Reply ID: %s", replyID))
	},
}

func init() {
	rootCmd.AddCommand(replyCmd)
}
