package presentation

import (
	"fmt"
	"time"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
)

// DisplayScheduledPostList prints the list of scheduled posts for a target.
// If fetchErr is non-nil, it prints the error instead.
func DisplayScheduledPostList(targetID string, posts []facebook.ScheduledPost, fetchErr error) {
	if fetchErr != nil {
		theme.Error(fetchErr.Error())
		return
	}

	theme.PrintSection(fmt.Sprintf("Scheduled Posts for %s", targetID))

	if len(posts) == 0 {
		theme.Info("Status", "No scheduled posts found.")
		return
	}

	for _, post := range posts {
		scheduledTime := "unknown"
		if post.ScheduledPublishTime > 0 {
			scheduledTime = time.Unix(post.ScheduledPublishTime, 0).Format("2006-01-02 15:04 MST")
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
		theme.Info("Message", msgPreview)
		theme.Info("Scheduled", scheduledTime)
	}

	theme.Info("Total", fmt.Sprintf("%d scheduled post(s)", len(posts)))
}
