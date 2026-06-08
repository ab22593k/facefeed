// Package presentation provides CLI output formatting for API results.
package presentation

import (
	"fmt"

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
		case statusDraft:
			draftCount++
		case statusScheduled:
			scheduledCount++
		case statusPublished:
			publishedCount++
		}

		timeInfo := FormatScheduledTime(post.ScheduledPublishTime)
		if timeInfo == statusUnknown && post.CreatedTime != "" {
			timeInfo = FormatCreatedTime(post.CreatedTime)
		}

		msgPreview := FormatMessage(post.Message)

		fmt.Printf("\n")
		theme.Info("Post ID", post.ID)
		theme.Info("Status", status)
		theme.Info("Message", msgPreview)

		if post.PermalinkURL != "" && status == statusPublished {
			theme.Info("Link", post.PermalinkURL)
		}

		label := "Scheduled"
		switch status {
		case statusDraft:
			label = "Created"
		case statusPublished:
			label = "Published"
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
		return statusPublished
	}
	if p.ScheduledPublishTime > 0 {
		return statusScheduled
	}
	return statusDraft
}
