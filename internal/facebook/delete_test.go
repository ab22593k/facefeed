package facebook

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDeletePostByID_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/post_123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	})
	defer server.Close()

	err := client.DeletePostByID("post_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePostByID_FacebookAPIError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "(#100) This post cannot be deleted",
				"code":    100,
			},
		})
	})
	defer server.Close()

	err := client.DeletePostByID("post_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to delete post") {
		t.Errorf("expected 'failed to delete post' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "This post cannot be deleted") {
		t.Errorf("expected FB error message in error, got: %v", err)
	}
}

func TestDeletePostByID_NetworkError(t *testing.T) {
	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	err := client.DeletePostByID("post_123")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	if !strings.Contains(err.Error(), "error sending delete request") {
		t.Errorf("expected 'error sending delete request' in error, got: %v", err)
	}
}
