package facebook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"facefeed/internal/testutil"
)

func TestDeletePostByID_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/post_123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		output := testutil.CaptureStdout(func() {
			DeletePostByID("post_123", "test_token")
		})

		if !strings.Contains(output, "Successfully deleted post: post_123") {
			t.Errorf("expected success message, got:\n%s", output)
		}
	})
}

func TestDeletePostByID_FacebookAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "(#100) This post cannot be deleted",
				"code":    100,
			},
		})
	}))
	defer server.Close()

	withFBClient(server, func() {
		output := testutil.CaptureStdout(func() {
			DeletePostByID("post_123", "test_token")
		})

		if !strings.Contains(output, "Failed to delete post") {
			t.Errorf("expected 'Failed to delete post' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "This post cannot be deleted") {
			t.Errorf("expected Facebook error message in output, got:\n%s", output)
		}
	})
}

func TestDeletePostByID_NetworkError(t *testing.T) {
	orig := GetClient()
	SetClient(&http.Client{Timeout: time.Millisecond})
	defer SetClient(orig)

	output := testutil.CaptureStdout(func() {
		DeletePostByID("post_123", "test_token")
	})

	if !strings.Contains(output, "Error sending delete request") {
		t.Errorf("expected 'Error sending delete request' in output, got:\n%s", output)
	}
}
