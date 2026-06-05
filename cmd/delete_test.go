package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeletePostByID_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/post_123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("access_token") != "test_token" {
			t.Errorf("expected access_token=test_token, got %s", r.URL.Query().Get("access_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		deletePostByID("post_123", "test_token")
	})

	if !strings.Contains(output, "Successfully deleted post: post_123") {
		t.Errorf("expected success message, got:\n%s", output)
	}
}

func TestDeletePostByID_FacebookAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "(#100) This post cannot be deleted",
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
		deletePostByID("post_123", "test_token")
	})

	if !strings.Contains(output, "Failed to delete post") {
		t.Errorf("expected 'Failed to delete post' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "This post cannot be deleted") {
		t.Errorf("expected Facebook error message in output, got:\n%s", output)
	}
	if !strings.Contains(output, "400") {
		t.Errorf("expected status code 400 in output, got:\n%s", output)
	}
}

func TestDeletePostByID_HTTPErrorWithoutMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		deletePostByID("post_123", "test_token")
	})

	if !strings.Contains(output, "Failed to delete post") {
		t.Errorf("expected 'Failed to delete post' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "500") {
		t.Errorf("expected status code 500 in output, got:\n%s", output)
	}
}

func TestDeletePostByID_NetworkError(t *testing.T) {
	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		deletePostByID("post_123", "test_token")
	})

	if !strings.Contains(output, "Error sending delete request") {
		t.Errorf("expected 'Error sending delete request' in output, got:\n%s", output)
	}
}

func TestDeletePostByID_InvalidPostID(t *testing.T) {
	// Test with a very long post ID to verify the URL is constructed correctly.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/abc_def_123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		deletePostByID("abc_def_123", "test_token")
	})

	if !strings.Contains(output, "Successfully deleted post: abc_def_123") {
		t.Errorf("expected success message with composite ID, got:\n%s", output)
	}
}

func TestDeletePostByID_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		deletePostByID("post_123", "test_token")
	})

	if !strings.Contains(output, "Failed to delete post") {
		t.Errorf("expected 'Failed to delete post' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "403") {
		t.Errorf("expected status code 403 in output, got:\n%s", output)
	}
}

func TestDeletePostByID_RequestCreation(t *testing.T) {
	// Verify that the function works even with unusual post ID characters.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()

	output := captureStdout(func() {
		deletePostByID("987654321_123456789", "test_token")
	})

	if !strings.Contains(output, "Successfully deleted post: 987654321_123456789") {
		t.Errorf("expected success message with composite ID, got:\n%s", output)
	}
}
