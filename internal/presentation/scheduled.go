package presentation

import (
	"fmt"

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
		scheduledTime := FormatScheduledTime(post.ScheduledPublishTime)
		msgPreview := FormatMessage(post.Message)

		fmt.Printf("\n")
		theme.Info("Post ID", post.ID)
		theme.Info("Message", msgPreview)
		theme.Info("Scheduled", scheduledTime)
	}

	theme.Info("Total", fmt.Sprintf("%d scheduled post(s)", len(posts)))
}
