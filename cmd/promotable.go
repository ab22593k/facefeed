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

// adsPost represents a single unpublished post from the ads_posts API response.
type adsPost struct {
	ID                   string `json:"id"`
	Message              string `json:"message"`
	IsPublished          bool   `json:"is_published"`
	ScheduledPublishTime int64  `json:"scheduled_publish_time"`
	CreatedTime          string `json:"created_time"`
	PermalinkURL         string `json:"permalink_url"`
}

// adsPostsResponse wraps the paginated API response.
type adsPostsResponse struct {
	Data   []adsPost `json:"data"`
	Paging *struct {
		Next string `json:"next"`
	} `json:"paging"`
}

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
			listAdsPosts(target.ID, envToken, limit)
		}
	},
}

func init() {
	rootCmd.AddCommand(promotableCmd)

	promotableCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to list unpublished posts for")
	promotableCmd.Flags().String("config", "", "Path to JSON config file for per-target listing")
	promotableCmd.Flags().Int("limit", 25, "Maximum number of unpublished posts to list (1-100)")
}

// postStatus returns a human-readable status string for an ads post.
func postStatus(p adsPost) string {
	if p.IsPublished {
		return "published"
	}
	if p.ScheduledPublishTime > 0 {
		return "scheduled"
	}
	return "draft"
}

// fetchAdsPosts makes the API request and returns the list of ads posts or an error.
func fetchAdsPosts(targetID, accessToken string, limit int) ([]adsPost, error) {
	fields := "id,message,is_published,scheduled_publish_time,created_time,permalink_url"
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/ads_posts?fields=%s&access_token=%s&limit=%d", targetID, fields, accessToken, limit)
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching unpublished posts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var fbErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &fbErr); err == nil && fbErr.Error.Message != "" {
			return nil, fmt.Errorf("Failed to fetch unpublished posts. Status: %s: %s", resp.Status, fbErr.Error.Message)
		}
		return nil, fmt.Errorf("Failed to fetch unpublished posts. Status: %s", resp.Status)
	}

	var result adsPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Error parsing response: %w", err)
	}

	return result.Data, nil
}

// displayAdsPosts prints the list of posts and a summary to the user.
func displayAdsPosts(targetID string, posts []adsPost) {
	theme.PrintSection(fmt.Sprintf("Unpublished Posts for %s", targetID))

	if len(posts) == 0 {
		theme.Info("Status", "No unpublished posts found.")
		return
	}

	// Count by status.
	draftCount := 0
	scheduledCount := 0
	publishedCount := 0

	for _, post := range posts {
		status := postStatus(post)

		switch status {
		case "draft":
			draftCount++
		case "scheduled":
			scheduledCount++
		case "published":
			publishedCount++
		}

		// Format timestamps.
		timeInfo := ""
		if post.ScheduledPublishTime > 0 {
			timeInfo = time.Unix(post.ScheduledPublishTime, 0).Format("2006-01-02 15:04 MST")
		} else if post.CreatedTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05-0700", post.CreatedTime); err == nil {
				timeInfo = t.Format("2006-01-02 15:04 MST")
			}
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
		theme.Info("Status", status)
		theme.Info("Message", msgPreview)

		if post.PermalinkURL != "" && status == "published" {
			theme.Info("Link", post.PermalinkURL)
		}

		label := "Scheduled"
		switch status {
		case "draft":
			label = "Created"
		case "published":
			label = "Published"
		}
		if timeInfo == "" {
			timeInfo = "unknown"
		}
		theme.Info(label, timeInfo)
	}

	// Show a breakdown summary.
	fmt.Println()
	theme.Info("Total", fmt.Sprintf("%d unpublished post(s)", len(posts)))
	if draftCount > 0 {
		theme.Info("Drafts", fmt.Sprintf("%d", draftCount))
	}
	if scheduledCount > 0 {
		theme.Info("Scheduled", fmt.Sprintf("%d", scheduledCount))
	}
	if publishedCount > 0 {
		theme.Info("Published", fmt.Sprintf("%d", publishedCount))
	}
}

// listAdsPosts is a thin wrapper around fetchAdsPosts + displayAdsPosts.
func listAdsPosts(targetID, accessToken string, limit int) {
	posts, err := fetchAdsPosts(targetID, accessToken, limit)
	if err != nil {
		theme.Error(err.Error())
		return
	}
	displayAdsPosts(targetID, posts)
}
