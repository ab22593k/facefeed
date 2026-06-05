package cmd

import (
	theme "bubble/internal"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// promotablePost represents a single post from the promotable_posts API response.
type promotablePost struct {
	ID                   string `json:"id"`
	Message              string `json:"message"`
	IsPublished          bool   `json:"is_published"`
	ScheduledPublishTime int64  `json:"scheduled_publish_time"`
	CreatedTime          string `json:"created_time"`
	PermalinkURL         string `json:"permalink_url"`
}

// promotablePostsResponse wraps the paginated API response.
type promotablePostsResponse struct {
	Data   []promotablePost `json:"data"`
	Paging *struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// promotableCmd represents the promotable command.
var promotableCmd = &cobra.Command{
	Use:   "promotable",
	Short: "List all promotable posts (unpublished, drafts, and scheduled)",
	Long: `List all promotable posts for one or more Facebook Pages.

Promotable posts include unpublished drafts, scheduled posts, and any other
posts that can be promoted as ads. Each post shows its ID, status (draft,
scheduled, or published), message preview, and timestamps.

Examples:
  bubble promotable
  bubble promotable --groups group_id
  bubble promotable --config targets.json
  bubble promotable --limit 50`,
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
			listPromotablePosts(target.ID, envToken, limit)
		}
	},
}

func init() {
	rootCmd.AddCommand(promotableCmd)

	promotableCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to list promotable posts for")
	promotableCmd.Flags().String("config", "", "Path to JSON config file for per-target listing")
	promotableCmd.Flags().Int("limit", 25, "Maximum number of promotable posts to list (1-100)")
}

// postStatus returns a human-readable status string for a promotable post.
func postStatus(p promotablePost) string {
	if p.IsPublished {
		return "published"
	}
	if p.ScheduledPublishTime > 0 {
		return "scheduled"
	}
	return "draft"
}

func listPromotablePosts(targetID, accessToken string, limit int) {
	theme.PrintSection(fmt.Sprintf("Promotable Posts for %s", targetID))

	fields := "id,message,is_published,scheduled_publish_time,created_time,permalink_url"
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/promotable_posts?is_published=false&fields=%s&access_token=%s&limit=%d", targetID, fields, accessToken, limit)
	resp, err := client.Get(apiURL)
	if err != nil {
		theme.Error(fmt.Sprintf("Error fetching promotable posts: %v", err))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		theme.Error(fmt.Sprintf("Failed to fetch promotable posts. Status: %s", resp.Status))
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

	var result promotablePostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		theme.Error(fmt.Sprintf("Error parsing response: %v", err))
		return
	}

	if len(result.Data) == 0 {
		theme.Info("Status", "No promotable posts found.")
		return
	}

	// Count by status.
	draftCount := 0
	scheduledCount := 0
	publishedCount := 0

	for _, post := range result.Data {
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
		if status == "draft" {
			label = "Created"
		} else if status == "published" {
			label = "Published"
		}
		if timeInfo == "" {
			timeInfo = "unknown"
		}
		theme.Info(label, timeInfo)
	}

	// Show a breakdown summary.
	fmt.Println()
	theme.Info("Total", fmt.Sprintf("%d promotable post(s)", len(result.Data)))
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
