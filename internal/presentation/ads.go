package presentation

import (
	"fmt"
	"time"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
)

// DisplayAdsPostList prints the list of ads (unpublished) posts for a target.
// If fetchErr is non-nil, it prints the error instead.
func DisplayAdsPostList(targetID string, posts []facebook.AdsPost, fetchErr error) {
	if fetchErr != nil {
		theme.Error(fetchErr.Error())
		return
	}

	theme.PrintSection(fmt.Sprintf("Unpublished Posts for %s", targetID))

	if len(posts) == 0 {
		theme.Info("Status", "No unpublished posts found.")
		return
	}

	draftCount := 0
	scheduledCount := 0
	publishedCount := 0

	for _, post := range posts {
		status := PostStatus(post)

		switch status {
		case "draft":
			draftCount++
		case "scheduled":
			scheduledCount++
		case "published":
			publishedCount++
		}

		timeInfo := ""
		if post.ScheduledPublishTime > 0 {
			timeInfo = time.Unix(post.ScheduledPublishTime, 0).Format("2006-01-02 15:04 MST")
		} else if post.CreatedTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05-0700", post.CreatedTime); err == nil {
				timeInfo = t.Format("2006-01-02 15:04 MST")
			}
		}

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

// PostStatus returns a human-readable status string for an ads post.
func PostStatus(p facebook.AdsPost) string {
	if p.IsPublished {
		return "published"
	}
	if p.ScheduledPublishTime > 0 {
		return "scheduled"
	}
	return "draft"
}
