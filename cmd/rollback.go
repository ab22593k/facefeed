package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
	"facefeed/internal/validation"

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

		theme.PrintSection("Rollback Batch")
		for _, target := range targets {
			theme.Info("Rolling back target", target.ID)
			rollbackTarget(FBClient, target.ID, specificPostID)
		}
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)

	rollbackCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to rollback from")
	rollbackCmd.Flags().String("config", "", "Path to JSON config file for per-target rollback")
}

func rollbackTarget(client facebook.Client, targetID, postID string) {
	targetPostID := postID

	if targetPostID == "" {
		theme.Info("Rollback", fmt.Sprintf("Fetching latest post for %s...", targetID))

		var err error
		targetPostID, err = client.FetchLatestFeedPost(targetID)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to fetch latest post: %v", err))
			return
		}
	}

	if err := client.DeletePostByID(targetPostID); err != nil {
		theme.Error(fmt.Sprintf("Failed to delete post %s: %v", targetPostID, err))
	} else {
		theme.Success(fmt.Sprintf("Deleted post %s", targetPostID))
	}
}
