package cmd

import (
	"os"

	theme "facefeed/internal"
	"facefeed/internal/presentation"

	"github.com/spf13/cobra"
)

// insightsCmd represents the insights command.
var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "View Page or Post insights",
	Long: `Retrieve Facebook Page or Post insights/metrics.

For Page insights, use --page-id or set FB_PAGE_ID:
  facefeed insights --metrics page_impressions,page_fan_adds --period day

For Post insights, use --post-id:
  facefeed insights --post-id 123456789_987654321 --metrics post_impressions,post_reactions_by_type_total

Common Page metrics:
  page_impressions, page_fan_adds, page_engaged_users, page_views_total

Common Post metrics:
  post_impressions, post_reactions_by_type_total, post_clicks, post_engaged_users

Period options: day, week, days_28 (for Page insights only)`,
	Run: func(cmd *cobra.Command, args []string) {
		envPageID := os.Getenv("FB_PAGE_ID")
		envToken := os.Getenv("FB_ACCESS_TOKEN")

		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		postID, _ := cmd.Flags().GetString("post-id")
		pageID, _ := cmd.Flags().GetString("page-id")
		metrics, _ := cmd.Flags().GetString("metrics")
		period, _ := cmd.Flags().GetString("period")

		objectID := postID
		if objectID == "" {
			objectID = pageID
			if objectID == "" {
				objectID = envPageID
			}
		}
		if objectID == "" {
			theme.Error("No target specified. Use --page-id, --post-id, or set FB_PAGE_ID.")
			os.Exit(1)
		}
		if metrics == "" {
			theme.Error("--metrics is required. Example: --metrics page_impressions,page_fan_adds")
			os.Exit(1)
		}

		label := "Post Insights"
		if postID == "" {
			label = "Page Insights"
		}
		theme.PrintSection(label)
		theme.Info("Object", objectID)
		theme.Info("Metrics", metrics)
		theme.Info("Period", period)

		data, err := FBClient.GetInsights(objectID, metrics, period)
		presentation.DisplayInsights(data, err)
	},
}

func init() {
	rootCmd.AddCommand(insightsCmd)

	insightsCmd.Flags().String("post-id", "", "Post ID to get insights for")
	insightsCmd.Flags().String("page-id", "", "Page ID to get insights for (defaults to FB_PAGE_ID)")
	insightsCmd.Flags().String("metrics", "", "Comma-separated list of insight metrics")
	insightsCmd.Flags().String("period", "day", "Time period: day, week, days_28")
}
