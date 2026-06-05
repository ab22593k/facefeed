package facebook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mustParseTime is a test helper that parses an RFC3339 time string or panics.
func mustParseTime(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(fmt.Sprintf("bad test time %q: %v", s, err))
	}
	return t.Unix()
}

// ---------------------------------------------------------------------------
// FetchAdsPosts tests
// ---------------------------------------------------------------------------

func TestFetchAdsPosts_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := adsPostsResponse{
			Data: []AdsPost{
				{ID: "p1", Message: "hello", IsPublished: false},
				{ID: "p2", Message: "world", IsPublished: false},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	posts, err := client.FetchAdsPosts("page_123", 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}
	if posts[0].ID != "p1" || posts[1].ID != "p2" {
		t.Errorf("unexpected post IDs: %v", posts)
	}
}

func TestFetchAdsPosts_NetworkError(t *testing.T) {
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.FetchAdsPosts("page_123", 25)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "Error fetching unpublished posts") {
		t.Errorf("expected 'Error fetching unpublished posts' in error, got: %v", err)
	}
}

func TestFetchAdsPosts_HTTPError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"Internal error"}}`))
	})
	defer server.Close()

	_, err := client.FetchAdsPosts("page_123", 25)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Failed to fetch") {
		t.Errorf("expected 'Failed to fetch' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

func TestFetchAdsPosts_InvalidJSON(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{garbage}`))
	})
	defer server.Close()

	_, err := client.FetchAdsPosts("page_123", 25)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Error parsing response") {
		t.Errorf("expected 'Error parsing response' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// FetchScheduledPosts tests
// ---------------------------------------------------------------------------

func TestFetchScheduledPosts_Success(t *testing.T) {
	scheduleTime := mustParseTime("2026-06-10T15:00:00Z")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := scheduledPostsResponse{
			Data: []ScheduledPost{
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
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	posts, err := client.FetchScheduledPosts("page_123", 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}
	if posts[0].ID != "page_123_001" {
		t.Errorf("expected ID page_123_001, got %s", posts[0].ID)
	}
}

func TestFetchScheduledPosts_NoPosts(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := scheduledPostsResponse{Data: []ScheduledPost{}}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	posts, err := client.FetchScheduledPosts("page_123", 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("got %d posts, want 0", len(posts))
	}
}

func TestFetchScheduledPosts_HTTPError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"Internal server error","code":1}}`))
	})
	defer server.Close()

	_, err := client.FetchScheduledPosts("page_123", 25)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Failed to fetch") {
		t.Errorf("expected 'Failed to fetch' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

func TestFetchScheduledPosts_NetworkError(t *testing.T) {
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.FetchScheduledPosts("page_123", 25)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "Error fetching scheduled posts") {
		t.Errorf("expected 'Error fetching scheduled posts' in error, got: %v", err)
	}
}
