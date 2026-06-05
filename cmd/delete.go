package cmd

import (
	theme "bubble/internal"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
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
  bubble delete 123456789_987654321
  bubble delete 987654321_123456789`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load()

		envToken := os.Getenv("FB_ACCESS_TOKEN")

		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		postID := args[0]
		deletePostByID(postID, envToken)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func deletePostByID(postID, accessToken string) {
	theme.PrintSection(fmt.Sprintf("Delete Post %s", postID))

	theme.Info("Post ID", postID)
	theme.Info("Status", "Deleting...")

	deleteURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s?access_token=%s", postID, accessToken)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		theme.Error(fmt.Sprintf("Error creating delete request: %v", err))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		theme.Error(fmt.Sprintf("Error sending delete request: %v", err))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		theme.Success(fmt.Sprintf("Successfully deleted post: %s", postID))
	} else {
		theme.Error(fmt.Sprintf("Failed to delete post. Status: %s", resp.Status))
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			theme.Error(errResp.Error.Message)
		}
	}
}
