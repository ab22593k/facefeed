package cmd

import (
	"encoding/json"
	theme "facefeed/internal"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// scheduledPost represents a single scheduled post from the API response.
type scheduledPost struct {
	ID                   string `json:"id"`
	Message              string `json:"message"`
	ScheduledPublishTime int64  `json:"scheduled_publish_time"`
	CreatedTime          string `json:"created_time"`
	Status               *struct {
		Published bool `json:"published"`
		Scheduled bool `json:"scheduled"`
	} `json:"status"`
}

// scheduledPostsResponse wraps the paginated API response.
type scheduledPostsResponse struct {
	Data   []scheduledPost `json:"data"`
	Paging *struct {
		Next string `json:"next"`
	} `json:"paging"`
}

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

		targets, err := resolveTargets(envPageID, groupsFlag, configPath)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to resolve targets: %v", err))
			os.Exit(1)
		}

		if len(targets) == 0 {
			theme.Error("No targets specified. Set FB_PAGE_ID, use --groups, or --config.")
			os.Exit(1)
		}

		for _, target := range targets {
			listScheduledPosts(target.ID, envToken, limit)
		}
	},
}

func init() {
	rootCmd.AddCommand(scheduledCmd)

	scheduledCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to list scheduled posts for")
	scheduledCmd.Flags().String("config", "", "Path to JSON config file for per-target listing")
	scheduledCmd.Flags().Int("limit", 25, "Maximum number of scheduled posts to list (1-100)")
}

func listScheduledPosts(targetID, accessToken string, limit int) {
	theme.PrintSection(fmt.Sprintf("Scheduled Posts for %s", targetID))

	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/scheduled_posts?access_token=%s&limit=%d", targetID, accessToken, limit)
	resp, err := client.Get(apiURL)
	if err != nil {
		theme.Error(fmt.Sprintf("Error fetching scheduled posts: %v", err))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		theme.Error(fmt.Sprintf("Failed to fetch scheduled posts. Status: %s", resp.Status))
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			theme.Error(errResp.Error.Message)
		}
		return
	}

	var result scheduledPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		theme.Error(fmt.Sprintf("Error parsing response: %v", err))
		return
	}

	if len(result.Data) == 0 {
		theme.Info("Status", "No scheduled posts found.")
		return
	}

	for _, post := range result.Data {
		// Format the scheduled time.
		scheduledTime := "unknown"
		if post.ScheduledPublishTime > 0 {
			scheduledTime = time.Unix(post.ScheduledPublishTime, 0).Format("2006-01-02 15:04 MST")
		}

		// Truncate message to a reasonable preview length.
		msgPreview := post.Message
		if len(msgPreview) > 80 {
			msgPreview = msgPreview[:80] + "..."
		}
		if msgPreview == "" {
			msgPreview = "(no message)"
		}

		fmt.Printf("\n")
		theme.Info("Post ID", post.ID)
		theme.Info("Message", msgPreview)
		theme.Info("Scheduled", scheduledTime)
	}

	// Show a count summary.
	theme.Info("Total", fmt.Sprintf("%d scheduled post(s)", len(result.Data)))
}
