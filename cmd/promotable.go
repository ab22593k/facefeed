package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
	"facefeed/internal/presentation"
	"facefeed/internal/validation"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// promotableCmd represents the promotable command.
var promotableCmd = &cobra.Command{
	Use:   "promotable",
	Short: "List all unpublished (dark) posts on a Page",
	Long: `List all unpublished (dark) posts for one or more Facebook Pages.

Dark posts are unpublished page posts — visible only in the Ads Manager
and via this API. They include drafts created in the Page composer or
via API calls with published=false. Each post shows its ID, status (draft
or scheduled), message preview, and timestamps.

Examples:
  facefeed promotable
  facefeed promotable --groups group_id
  facefeed promotable --config targets.json
  facefeed promotable --limit 50`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load()

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
			posts, err := facebook.FetchAdsPosts(target.ID, envToken, limit)
			presentation.DisplayAdsPostList(target.ID, posts, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(promotableCmd)

	promotableCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to list unpublished posts for")
	promotableCmd.Flags().String("config", "", "Path to JSON config file for per-target listing")
	promotableCmd.Flags().Int("limit", 25, "Maximum number of unpublished posts to list (1-100)")
}
