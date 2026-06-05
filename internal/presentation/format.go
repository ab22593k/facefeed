package presentation

import (
	"time"
)

const maxMessageLen = 80

// FormatMessage truncates a message to a preview length, appending "..." if truncated.
// Returns "(no message)" for empty strings.
func FormatMessage(msg string) string {
	if msg == "" {
		return "(no message)"
	}
	if len(msg) > maxMessageLen {
		return msg[:maxMessageLen] + "..."
	}
	return msg
}

// FormatScheduledTime formats a Unix timestamp as "2006-01-02 15:04 MST".
// Returns "unknown" if ts is 0 or negative.
func FormatScheduledTime(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04 MST")
}

// FormatCreatedTime parses and formats a Facebook created_time string.
// Returns "unknown" if the string cannot be parsed.
func FormatCreatedTime(createdTime string) string {
	if createdTime == "" {
		return "unknown"
	}
	t, err := time.Parse("2006-01-02T15:04:05-0700", createdTime)
	if err != nil {
		return "unknown"
	}
	return t.Format("2006-01-02 15:04 MST")
}
