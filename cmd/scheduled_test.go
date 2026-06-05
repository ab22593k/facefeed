package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
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

func TestListScheduledPosts_Success(t *testing.T) {

	// Set up a mock server that returns scheduled posts.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/scheduled_posts") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "25" {
			t.Errorf("expected limit=25, got %s", r.URL.Query().Get("limit"))
		}

		scheduleTime := mustParseTime("2026-06-10T15:00:00Z")

		resp := scheduledPostsResponse{
			Data: []scheduledPost{
				{
					ID:                   "page_123_001",
					Message:              "Hello from the future!",
					ScheduledPublishTime: scheduleTime,
					CreatedTime:          "2026-06-01T10:00:00+0000",
					Status: &struct {
						Published bool `json:"published"`
						Scheduled bool `json:"scheduled"`
					}{Published: false, Scheduled: true},
				},
				{
					ID:                   "page_123_002",
					Message:              "Another scheduled post with a message",
					ScheduledPublishTime: scheduleTime + 3600,
					Status: &struct {
						Published bool `json:"published"`
						Scheduled bool `json:"scheduled"`
					}{Published: false, Scheduled: true},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Override the global client to use the test server.
	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	// Capture stdout.
	output := captureStdout(func() {
		listScheduledPosts("page_123", "test_token", 25)
	})

	// Verify output contains expected information.
	if !strings.Contains(output, "page_123_001") {
		t.Errorf("output missing post ID 'page_123_001'\n%s", output)
	}
	if !strings.Contains(output, "page_123_002") {
		t.Errorf("output missing post ID 'page_123_002'\n%s", output)
	}
	if !strings.Contains(output, "Hello from the future!") {
		t.Errorf("output missing message preview\n%s", output)
	}
	if !strings.Contains(output, "2026-06-10") {
		t.Errorf("output missing scheduled date\n%s", output)
	}
	// Just verify the date appears with a time pattern. Timezone varies by machine.
	if !regexp.MustCompile(`2026-06-10 \d{2}:\d{2} [A-Z]{2,5}`).MatchString(output) {
		t.Errorf("output missing expected time pattern (date + HH:MM TZ)\n%s", output)
	}
	if !strings.Contains(output, "2 scheduled post(s)") {
		t.Errorf("output missing count summary\n%s", output)
	}
}

func TestListScheduledPosts_NoPosts(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := scheduledPostsResponse{Data: []scheduledPost{}}
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
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "No scheduled posts found") {
		t.Errorf("expected 'No scheduled posts found', got:\n%s", output)
	}
}

func TestListScheduledPosts_HTTPError(t *testing.T) {

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
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "Failed to fetch") {
		t.Errorf("expected 'Failed to fetch' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "500") {
		t.Errorf("expected status code 500 in output, got:\n%s", output)
	}
}

func TestListScheduledPosts_FacebookAPIError(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "(#100) Insufficient permission to view scheduled posts",
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
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "Insufficient permission") {
		t.Errorf("expected Facebook error message in output, got:\n%s", output)
	}
}

func TestListScheduledPosts_NetworkError(t *testing.T) {

	// Point client at an unreachable address.
	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "Error fetching scheduled posts") {
		t.Errorf("expected 'Error fetching scheduled posts' in output, got:\n%s", output)
	}
}

func TestListScheduledPosts_EmptyBodyJSONError(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "Error parsing response") {
		t.Errorf("expected 'Error parsing response' in output, got:\n%s", output)
	}
}

func TestListScheduledPosts_LongMessageTruncation(t *testing.T) {

	longMsg := strings.Repeat("A", 100)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := scheduledPostsResponse{
			Data: []scheduledPost{
				{
					ID:                   "page_123_trunc",
					Message:              longMsg,
					ScheduledPublishTime: mustParseTime("2026-07-01T12:00:00Z"),
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
		listScheduledPosts("page_123", "test_token", 25)
	})

	// Message should be truncated to 80 chars + "...".
	expectedPreview := strings.Repeat("A", 80) + "..."
	if !strings.Contains(output, expectedPreview) {
		t.Errorf("expected truncated message in output:\n%s", output)
	}
	if strings.Contains(output, longMsg) {
		t.Error("output contains untruncated message")
	}
}

func TestListScheduledPosts_EmptyMessage(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := scheduledPostsResponse{
			Data: []scheduledPost{
				{
					ID:                   "page_123_empty",
					Message:              "",
					ScheduledPublishTime: mustParseTime("2026-07-01T12:00:00Z"),
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
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "(no message)") {
		t.Errorf("expected '(no message)' placeholder in output, got:\n%s", output)
	}
}

func TestListScheduledPosts_NoScheduledTime(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := scheduledPostsResponse{
			Data: []scheduledPost{
				{
					ID:                   "page_123_notime",
					Message:              "Post without schedule time",
					ScheduledPublishTime: 0, // No schedule time set
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
		listScheduledPosts("page_123", "test_token", 25)
	})

	if !strings.Contains(output, "unknown") {
		t.Errorf("expected 'unknown' for missing schedule time, got:\n%s", output)
	}
}

// rewriteTransport is an http.RoundTripper that rewrites the host in requests
// so they can be directed to a test server while preserving the original URL path.
type rewriteTransport struct {
	origin string
	target string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request with the host rewritten.
	clone := req.Clone(req.Context())
	clone.URL.Host = rt.target
	clone.URL.Scheme = "http"
	clone.Host = rt.target
	return http.DefaultTransport.RoundTrip(clone)
}

// captureStdout executes fn and returns everything written to stdout as a string.
func captureStdout(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(fmt.Sprintf("failed to create pipe: %v", err))
	}

	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}
