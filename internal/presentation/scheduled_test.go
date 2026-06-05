package presentation

import (
	"strings"
	"testing"

	"facefeed/internal/facebook"
	"facefeed/internal/testutil"
)

func TestDisplayScheduledPostList_Error(t *testing.T) {
	output := testutil.CaptureStdout(func() {
		DisplayScheduledPostList("page_123", nil, assertError("fetch failed"))
	})

	if !strings.Contains(output, "fetch failed") {
		t.Errorf("expected error message in output, got:\n%s", output)
	}
}

func TestDisplayScheduledPostList_Empty(t *testing.T) {
	output := testutil.CaptureStdout(func() {
		DisplayScheduledPostList("page_123", []facebook.ScheduledPost{}, nil)
	})

	if !strings.Contains(output, "No scheduled posts found") {
		t.Errorf("expected 'No scheduled posts found', got:\n%s", output)
	}
}

func TestDisplayScheduledPostList_Success(t *testing.T) {
	scheduleTime := mustParseTime("2026-06-10T15:00:00Z")

	posts := []facebook.ScheduledPost{
		{
			ID:                   "page_123_001",
			Message:              "Hello from the future!",
			ScheduledPublishTime: scheduleTime,
		},
		{
			ID:                   "page_123_002",
			Message:              "Another scheduled post",
			ScheduledPublishTime: scheduleTime + 3600,
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayScheduledPostList("page_123", posts, nil)
	})

	if !strings.Contains(output, "page_123_001") {
		t.Errorf("output missing post ID\n%s", output)
	}
	if !strings.Contains(output, "page_123_002") {
		t.Errorf("output missing post ID\n%s", output)
	}
	if !strings.Contains(output, "Hello from the future!") {
		t.Errorf("output missing message\n%s", output)
	}
	if !strings.Contains(output, "2 scheduled post(s)") {
		t.Errorf("output missing count summary\n%s", output)
	}
}

func TestDisplayScheduledPostList_LongMessage(t *testing.T) {
	longMsg := strings.Repeat("C", 100)
	posts := []facebook.ScheduledPost{
		{
			ID:                   "long_1",
			Message:              longMsg,
			ScheduledPublishTime: mustParseTime("2026-08-01T12:00:00Z"),
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayScheduledPostList("page_123", posts, nil)
	})

	expectedPreview := strings.Repeat("C", 80) + "..."
	if !strings.Contains(output, expectedPreview) {
		t.Errorf("expected truncated message in output:\n%s", output)
	}
}

func TestDisplayScheduledPostList_EmptyMessage(t *testing.T) {
	posts := []facebook.ScheduledPost{
		{
			ID:                   "empty_1",
			Message:              "",
			ScheduledPublishTime: mustParseTime("2026-08-01T12:00:00Z"),
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayScheduledPostList("page_123", posts, nil)
	})

	if !strings.Contains(output, "(no message)") {
		t.Errorf("expected '(no message)' placeholder, got:\n%s", output)
	}
}
