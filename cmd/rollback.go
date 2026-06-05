package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
	"facefeed/internal/validation"

	"facefeed/domain"

	"github.com/spf13/cobra"
)

// rollbackCmd represents the rollback command.
var rollbackCmd = &cobra.Command{
	Use:   "rollback [post_id]",
	Short: "Delete a Facebook post",
	Long: `Delete a specific post by its ID, or the latest post if no ID is provided.
If multiple targets are configured, the post is deleted from all of them.

Examples:
  facefeed rollback
  facefeed rollback 123456789_987654321`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envPageID := os.Getenv("FB_PAGE_ID")
		envToken := os.Getenv("FB_ACCESS_TOKEN")
		groupsFlag, _ := cmd.Flags().GetString("groups")
		configPath, _ := cmd.Flags().GetString("config")

		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		targets, err := validation.ResolveTargets(envPageID, groupsFlag, configPath)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to resolve targets: %v", err))
			os.Exit(1)
		}

		if len(targets) == 0 {
			theme.Error("No targets specified. Set FB_PAGE_ID, use --groups, or --config.")
			os.Exit(1)
		}

		specificPostID := ""
		if len(args) > 0 {
			specificPostID = args[0]
		}

		rollbackPosts(targets, specificPostID, envToken)
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)

	rollbackCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to rollback from")
	rollbackCmd.Flags().String("config", "", "Path to JSON config file for per-target rollback")
}

func rollbackPosts(targets []domain.PublishTarget, specificPostID, accessToken string) {
	theme.PrintSection("Rollback Batch")
	for _, target := range targets {
		theme.Info("Rolling back target", target.ID)
		rollbackTarget(target.ID, specificPostID, accessToken)
	}
}

func rollbackTarget(targetID, postID, accessToken string) {
	targetPostID := postID

	if targetPostID == "" {
		theme.Info("Rollback", fmt.Sprintf("Fetching latest post for %s...", targetID))

		apiURL := facebook.GraphAPIURL(targetID+"/feed") + fmt.Sprintf("?limit=1&access_token=%s", accessToken)
		resp, err := facebook.GetClient().Get(apiURL)
		if err != nil {
			theme.Error(fmt.Sprintf("Error fetching feed: %v", err))
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			theme.Error(fmt.Sprintf("Failed to fetch feed. Status: %s, Body: %s", resp.Status, string(body)))
			return
		}

		var feed struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &feed); err != nil {
			theme.Error(fmt.Sprintf("Error parsing feed JSON: %v", err))
			return
		}

		if len(feed.Data) == 0 {
			theme.Warning(fmt.Sprintf("No posts found for target %s.", targetID))
			return
		}

		targetPostID = feed.Data[0].ID
	}

	facebook.DeletePostByID(targetPostID, accessToken)
}
