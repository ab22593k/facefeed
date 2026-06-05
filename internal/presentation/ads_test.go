package presentation

import (
	"strings"
	"testing"
	"time"

	"facefeed/internal/facebook"
	"facefeed/internal/testutil"
)

// ---------------------------------------------------------------------------
// PostStatus tests
// ---------------------------------------------------------------------------

func TestPostStatus_Draft(t *testing.T) {
	p := facebook.AdsPost{ID: "p1", IsPublished: false, ScheduledPublishTime: 0}
	if got := PostStatus(p); got != "draft" {
		t.Errorf("PostStatus(draft) = %q, want %q", got, "draft")
	}
}

func TestPostStatus_Scheduled(t *testing.T) {
	p := facebook.AdsPost{ID: "p2", IsPublished: false, ScheduledPublishTime: 2000000000}
	if got := PostStatus(p); got != "scheduled" {
		t.Errorf("PostStatus(scheduled) = %q, want %q", got, "scheduled")
	}
}

func TestPostStatus_Published(t *testing.T) {
	p := facebook.AdsPost{ID: "p3", IsPublished: true, ScheduledPublishTime: 0}
	if got := PostStatus(p); got != "published" {
		t.Errorf("PostStatus(published) = %q, want %q", got, "published")
	}
}

// ---------------------------------------------------------------------------
// DisplayAdsPostList tests
// ---------------------------------------------------------------------------

func TestDisplayAdsPostList_Error(t *testing.T) {
	output := testutil.CaptureStdout(func() {
		DisplayAdsPostList("page_123", nil, assertError("fetch failed"))
	})

	if !strings.Contains(output, "fetch failed") {
		t.Errorf("expected error message in output, got:\n%s", output)
	}
}

func TestDisplayAdsPostList_Empty(t *testing.T) {
	output := testutil.CaptureStdout(func() {
		DisplayAdsPostList("page_123", []facebook.AdsPost{}, nil)
	})

	if !strings.Contains(output, "No unpublished posts found") {
		t.Errorf("expected 'No unpublished posts found', got:\n%s", output)
	}
}

func TestDisplayAdsPostList_Mixed(t *testing.T) {
	scheduleTime := mustParseTime("2026-06-15T14:00:00Z")
	now := time.Now()

	posts := []facebook.AdsPost{
		{
			ID:          "page_456_draft1",
			Message:     "Draft post in progress",
			IsPublished: false,
			CreatedTime: "2026-06-01T09:00:00+0000",
		},
		{
			ID:                   "page_456_sched1",
			Message:              "Scheduled announcement",
			IsPublished:          false,
			ScheduledPublishTime: scheduleTime,
			CreatedTime:          "2026-06-01T10:00:00+0000",
		},
		{
			ID:          "page_456_pub1",
			Message:     "Published post!",
			IsPublished: true,
			CreatedTime: now.Format("2006-01-02T15:04:05-0700"),
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayAdsPostList("page_456", posts, nil)
	})

	if !strings.Contains(output, "page_456_draft1") {
		t.Errorf("output missing draft post ID\n%s", output)
	}
	if !strings.Contains(output, "page_456_sched1") {
		t.Errorf("output missing scheduled post ID\n%s", output)
	}
	if !strings.Contains(output, "draft") {
		t.Errorf("output missing 'draft' status\n%s", output)
	}
	if !strings.Contains(output, "scheduled") {
		t.Errorf("output missing 'scheduled' status\n%s", output)
	}
	if !strings.Contains(output, "3 unpublished post(s)") {
		t.Errorf("output missing count summary\n%s", output)
	}
	if !strings.Contains(output, "Drafts") {
		t.Errorf("output missing Drafts count\n%s", output)
	}
	if !strings.Contains(output, "Scheduled") {
		t.Errorf("output missing Scheduled count\n%s", output)
	}
	if !strings.Contains(output, "Published") {
		t.Errorf("output missing Published count\n%s", output)
	}
}

func TestDisplayAdsPostList_LongMessage(t *testing.T) {
	longMsg := strings.Repeat("C", 100)
	posts := []facebook.AdsPost{
		{
			ID:                   "long_1",
			Message:              longMsg,
			IsPublished:          false,
			ScheduledPublishTime: mustParseTime("2026-08-01T12:00:00Z"),
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayAdsPostList("page_123", posts, nil)
	})

	expectedPreview := strings.Repeat("C", 80) + "..."
	if !strings.Contains(output, expectedPreview) {
		t.Errorf("expected truncated message in output:\n%s", output)
	}
}

func TestDisplayAdsPostList_EmptyMessage(t *testing.T) {
	posts := []facebook.AdsPost{
		{
			ID:                   "empty_1",
			Message:              "",
			IsPublished:          false,
			ScheduledPublishTime: mustParseTime("2026-08-01T12:00:00Z"),
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayAdsPostList("page_123", posts, nil)
	})

	if !strings.Contains(output, "(no message)") {
		t.Errorf("expected '(no message)' placeholder, got:\n%s", output)
	}
}

func TestDisplayAdsPostList_PermalinkForPublished(t *testing.T) {
	posts := []facebook.AdsPost{
		{
			ID:           "pub_1",
			Message:      "Published post",
			IsPublished:  true,
			PermalinkURL: "https://fb.com/post/123",
		},
	}

	output := testutil.CaptureStdout(func() {
		DisplayAdsPostList("page_123", posts, nil)
	})

	if !strings.Contains(output, "https://fb.com/post/123") {
		t.Errorf("expected permalink for published post, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// assertError returns an error with the given message, for use in error-display tests.
func assertError(msg string) error {
	return &testError{msg}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// mustParseTime is a test helper that parses an RFC3339 time string or panics.
func mustParseTime(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic("bad test time " + s + ": " + err.Error())
	}
	return t.Unix()
}
