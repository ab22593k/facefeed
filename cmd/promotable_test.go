package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// postStatus helper tests
// ---------------------------------------------------------------------------

func TestPostStatus_Draft(t *testing.T) {
	p := adsPost{ID: "p1", IsPublished: false, ScheduledPublishTime: 0}
	if got := postStatus(p); got != "draft" {
		t.Errorf("postStatus(draft) = %q, want %q", got, "draft")
	}
}

func TestPostStatus_Scheduled(t *testing.T) {
	p := adsPost{ID: "p2", IsPublished: false, ScheduledPublishTime: 2000000000}
	if got := postStatus(p); got != "scheduled" {
		t.Errorf("postStatus(scheduled) = %q, want %q", got, "scheduled")
	}
}

func TestPostStatus_Published(t *testing.T) {
	p := adsPost{ID: "p3", IsPublished: true, ScheduledPublishTime: 0}
	if got := postStatus(p); got != "published" {
		t.Errorf("postStatus(published) = %q, want %q", got, "published")
	}
}

func TestPostStatus_PublishedWithScheduleTime(t *testing.T) {
	// When IsPublished is true, it should return "published" regardless of schedule time.
	p := adsPost{ID: "p4", IsPublished: true, ScheduledPublishTime: 2000000000}
	if got := postStatus(p); got != "published" {
		t.Errorf("postStatus(published with schedule) = %q, want %q", got, "published")
	}
}

// ---------------------------------------------------------------------------
// listAdsPosts tests
// ---------------------------------------------------------------------------

func TestListAdsPosts_Success_Mixed(t *testing.T) {
	scheduleTime := mustParseTime("2026-06-15T14:00:00Z")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/ads_posts") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "25" {
			t.Errorf("expected limit=25, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("fields") == "" {
			t.Error("expected fields parameter")
		}

		resp := adsPostsResponse{
			Data: []adsPost{
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
			},
			Paging: nil,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	// Check post IDs.
	if !strings.Contains(output, "page_456_draft1") {
		t.Errorf("output missing draft post ID\n%s", output)
	}
	if !strings.Contains(output, "page_456_sched1") {
		t.Errorf("output missing scheduled post ID\n%s", output)
	}

	// Check status labels.
	if !strings.Contains(output, "draft") {
		t.Errorf("output missing 'draft' status\n%s", output)
	}
	if !strings.Contains(output, "scheduled") {
		t.Errorf("output missing 'scheduled' status\n%s", output)
	}

	// Check messages.
	if !strings.Contains(output, "Draft post in progress") {
		t.Errorf("output missing draft message\n%s", output)
	}
	if !strings.Contains(output, "Scheduled announcement") {
		t.Errorf("output missing scheduled message\n%s", output)
	}

	// Check scheduled time formatted.
	if !regexp.MustCompile(`2026-06-15 \d{2}:\d{2} [A-Z]{2,5}`).MatchString(output) {
		t.Errorf("output missing expected scheduled time pattern\n%s", output)
	}

	// Check created time for draft (should show as "Created" with date).
	if !strings.Contains(output, "2026-06-01") {
		t.Errorf("output missing created date for draft\n%s", output)
	}

	// Check summary counts.
	if !strings.Contains(output, "2 unpublished post(s)") {
		t.Errorf("output missing count summary\n%s", output)
	}
	if !strings.Contains(output, "Drafts") {
		t.Errorf("output missing Drafts count\n%s", output)
	}
	if !strings.Contains(output, "Scheduled") {
		t.Errorf("output missing Scheduled count\n%s", output)
	}
}

func TestListAdsPosts_NoPosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{Data: []adsPost{}}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "No unpublished posts found") {
		t.Errorf("expected 'No unpublished posts found', got:\n%s", output)
	}
}

func TestListAdsPosts_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"Internal server error","code":1}}`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "Failed to fetch") {
		t.Errorf("expected 'Failed to fetch' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "500") {
		t.Errorf("expected status code 500 in output, got:\n%s", output)
	}
}

func TestListAdsPosts_FacebookAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "(#100) Invalid parameter",
				"code":    100,
				"type":    "OAuthException",
			},
		})
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "Invalid parameter") {
		t.Errorf("expected Facebook error message in output, got:\n%s", output)
	}
}

func TestListAdsPosts_NetworkError(t *testing.T) {
	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "Error fetching unpublished posts") {
		t.Errorf("expected 'Error fetching unpublished posts' in output, got:\n%s", output)
	}
}

func TestListAdsPosts_LongMessageTruncation(t *testing.T) {
	longMsg := strings.Repeat("B", 100)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{
			Data: []adsPost{
				{
					ID:          "page_long_1",
					Message:     longMsg,
					IsPublished: false,
					CreatedTime: "2026-06-01T10:00:00+0000",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	// Message should be truncated to 80 chars + "...".
	expectedPreview := strings.Repeat("B", 80) + "..."
	if !strings.Contains(output, expectedPreview) {
		t.Errorf("expected truncated message in output:\n%s", output)
	}
	if strings.Contains(output, longMsg) {
		t.Error("output contains untruncated message")
	}
}

func TestListAdsPosts_EmptyMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{
			Data: []adsPost{
				{
					ID:          "page_empty_1",
					Message:     "",
					IsPublished: false,
					CreatedTime: "2026-06-01T10:00:00+0000",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "(no message)") {
		t.Errorf("expected '(no message)' placeholder in output, got:\n%s", output)
	}
}

func TestListAdsPosts_CreatedTimeFallback(t *testing.T) {
	// Post has no ScheduledPublishTime, but has CreatedTime — should show "Created" label with date.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{
			Data: []adsPost{
				{
					ID:                   "page_ct_1",
					Message:              "Post with created time",
					IsPublished:          false,
					ScheduledPublishTime: 0,
					CreatedTime:          "2026-06-01T10:00:00+0000",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "Created") {
		t.Errorf("expected 'Created' label for draft post, got:\n%s", output)
	}
	if !strings.Contains(output, "2026-06-01") {
		t.Errorf("expected created date in output, got:\n%s", output)
	}
}

func TestListAdsPosts_UnknownTime(t *testing.T) {
	// Post has no ScheduledPublishTime and no CreatedTime — should show "unknown".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{
			Data: []adsPost{
				{
					ID:                   "page_notime_1",
					Message:              "No time info",
					IsPublished:          false,
					ScheduledPublishTime: 0,
					CreatedTime:          "",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "unknown") {
		t.Errorf("expected 'unknown' for missing time info, got:\n%s", output)
	}
}

func TestListAdsPosts_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listAdsPosts("page_456", "test_token", 25)
	})

	if !strings.Contains(output, "Error parsing response") {
		t.Errorf("expected 'Error parsing response' in output, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// fetchAdsPosts direct tests (no stdout capture needed)
// ---------------------------------------------------------------------------

func TestFetchAdsPosts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{
			Data: []adsPost{
				{ID: "p1", Message: "hello", IsPublished: false},
				{ID: "p2", Message: "world", IsPublished: false},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	withTestClient(server, func() {
		posts, err := fetchAdsPosts("page_123", "tok", 25)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(posts) != 2 {
			t.Fatalf("got %d posts, want 2", len(posts))
		}
		if posts[0].ID != "p1" || posts[1].ID != "p2" {
			t.Errorf("unexpected post IDs: %v", posts)
		}
	})
}

func TestFetchAdsPosts_NetworkError(t *testing.T) {
	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	_, err := fetchAdsPosts("page_123", "tok", 25)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "Error fetching unpublished posts") {
		t.Errorf("expected 'Error fetching unpublished posts' in error, got: %v", err)
	}
}

func TestFetchAdsPosts_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"Internal error"}}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		_, err := fetchAdsPosts("page_123", "tok", 25)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Failed to fetch") {
			t.Errorf("expected 'Failed to fetch' in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected '500' in error, got: %v", err)
		}
		// Facebook error message should be included.
		if !strings.Contains(err.Error(), "Internal error") {
			t.Errorf("expected Facebook error message in error, got: %v", err)
		}
	})
}

func TestFetchAdsPosts_HTTPErrorNoFBMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		_, err := fetchAdsPosts("page_123", "tok", 25)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Failed to fetch") {
			t.Errorf("expected 'Failed to fetch' in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "502") {
			t.Errorf("expected '502' in error, got: %v", err)
		}
	})
}

func TestFetchAdsPosts_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{garbage}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		_, err := fetchAdsPosts("page_123", "tok", 25)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Error parsing response") {
			t.Errorf("expected 'Error parsing response' in error, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// displayAdsPosts direct tests (no httptest needed)
// ---------------------------------------------------------------------------

func TestDisplayAdsPosts_Empty(t *testing.T) {
	output := captureStdout(func() {
		displayAdsPosts("page_123", []adsPost{})
	})

	if !strings.Contains(output, "No unpublished posts found") {
		t.Errorf("expected 'No unpublished posts found', got:\n%s", output)
	}
}

func TestDisplayAdsPosts_LongMessage(t *testing.T) {
	longMsg := strings.Repeat("C", 100)
	posts := []adsPost{
		{
			ID:                   "long_1",
			Message:              longMsg,
			IsPublished:          false,
			ScheduledPublishTime: mustParseTime("2026-08-01T12:00:00Z"),
		},
	}

	output := captureStdout(func() {
		displayAdsPosts("page_123", posts)
	})

	expectedPreview := strings.Repeat("C", 80) + "..."
	if !strings.Contains(output, expectedPreview) {
		t.Errorf("expected truncated message in output:\n%s", output)
	}
}

func TestDisplayAdsPosts_EmptyMessage(t *testing.T) {
	posts := []adsPost{
		{
			ID:                   "empty_1",
			Message:              "",
			IsPublished:          false,
			ScheduledPublishTime: mustParseTime("2026-08-01T12:00:00Z"),
		},
	}

	output := captureStdout(func() {
		displayAdsPosts("page_123", posts)
	})

	if !strings.Contains(output, "(no message)") {
		t.Errorf("expected '(no message)' placeholder, got:\n%s", output)
	}
}

func TestDisplayAdsPosts_UnknownTime(t *testing.T) {
	posts := []adsPost{
		{
			ID:                   "notime_1",
			Message:              "No time",
			IsPublished:          false,
			ScheduledPublishTime: 0,
			CreatedTime:          "",
		},
	}

	output := captureStdout(func() {
		displayAdsPosts("page_123", posts)
	})

	if !strings.Contains(output, "unknown") {
		t.Errorf("expected 'unknown' for missing time, got:\n%s", output)
	}
}

func TestDisplayAdsPosts_MixedCounts(t *testing.T) {
	posts := []adsPost{
		{ID: "d1", Message: "draft", IsPublished: false, ScheduledPublishTime: 0},
		{ID: "s1", Message: "scheduled", IsPublished: false, ScheduledPublishTime: 2000000000},
		{ID: "p1", Message: "published", IsPublished: true, ScheduledPublishTime: 0},
	}

	output := captureStdout(func() {
		displayAdsPosts("page_123", posts)
	})

	if !strings.Contains(output, "3 unpublished post(s)") {
		t.Errorf("expected '3 unpublished post(s)', got:\n%s", output)
	}
	if !strings.Contains(output, "Drafts") {
		t.Errorf("expected Drafts count, got:\n%s", output)
	}
	if !strings.Contains(output, "Scheduled") {
		t.Errorf("expected Scheduled count, got:\n%s", output)
	}
	if !strings.Contains(output, "Published") {
		t.Errorf("expected Published count, got:\n%s", output)
	}
}

func TestDisplayAdsPosts_PermalinkForPublished(t *testing.T) {
	posts := []adsPost{
		{
			ID:           "pub_1",
			Message:      "Published post",
			IsPublished:  true,
			PermalinkURL: "https://fb.com/post/123",
		},
	}

	output := captureStdout(func() {
		displayAdsPosts("page_123", posts)
	})

	if !strings.Contains(output, "https://fb.com/post/123") {
		t.Errorf("expected permalink for published post, got:\n%s", output)
	}
}

func TestDisplayAdsPosts_NoPermalinkForDraft(t *testing.T) {
	posts := []adsPost{
		{
			ID:           "draft_1",
			Message:      "Draft post",
			IsPublished:  false,
			PermalinkURL: "https://fb.com/post/456",
		},
	}

	output := captureStdout(func() {
		displayAdsPosts("page_123", posts)
	})

	// Permalink should NOT be shown for non-published posts.
	if strings.Contains(output, "https://fb.com/post/456") {
		t.Errorf("expected no permalink for draft post, got:\n%s", output)
	}
}
