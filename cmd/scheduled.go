package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
	"facefeed/internal/presentation"
	"facefeed/internal/validation"

	"github.com/spf13/cobra"
)

// scheduledCmd represents the scheduled command.
var scheduledCmd = &cobra.Command{
	Use:   "scheduled",
	Short: "List scheduled/pending posts for a page",
	Long: `List all scheduled and pending posts for one or more Facebook Pages.

Scheduled posts are posts created with --schedule that have not yet been published.
The list shows the post ID, message preview, and scheduled time for each post.

Examples:
  facefeed scheduled
  facefeed scheduled --groups group_id
  facefeed scheduled --config targets.json`,
	Run: func(cmd *cobra.Command, args []string) {
		envPageID := os.Getenv("FB_PAGE_ID")
		envToken := os.Getenv("FB_ACCESS_TOKEN")
		groupsFlag, _ := cmd.Flags().GetString("groups")
		configPath, _ := cmd.Flags().GetString("config")
		limit, _ := cmd.Flags().GetInt("limit")
		if limit < 1 || limit > 100 {
			theme.Error("--limit must be between 1 and 100.")
			os.Exit(1)
		}

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

		for _, target := range targets {
			posts, err := facebook.FetchScheduledPosts(target.ID, envToken, limit)
			presentation.DisplayScheduledPostList(target.ID, posts, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(scheduledCmd)

	scheduledCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to list scheduled posts for")
	scheduledCmd.Flags().String("config", "", "Path to JSON config file for per-target listing")
	scheduledCmd.Flags().Int("limit", 25, "Maximum number of scheduled posts to list (1-100)")
}
