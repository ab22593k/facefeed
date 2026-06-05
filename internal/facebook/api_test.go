package facebook

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"facefeed/domain"
	"facefeed/internal/testutil"
)

// ---------------------------------------------------------------------------
// postText tests
// ---------------------------------------------------------------------------

func TestPostText_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/feed")
		testutil.AssertFormValue(t, r, "message", "Hello, world!")
		testutil.AssertFormValue(t, r, "access_token", "tok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "123_456"}`))
	})
	defer server.Close()

	id, err := client.PostText("page1", "Hello, world!", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "123_456" {
		t.Errorf("got id %q, want %q", id, "123_456")
	}
}

func TestPostText_WithTargeting(t *testing.T) {
	targeting := `{"geo_locations":{"countries":["US"]}}`

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertFormValue(t, r, "targeting", targeting)
		testutil.AssertFormValue(t, r, "message", "Targeted post")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "789"}`))
	})
	defer server.Close()

	id, err := client.PostText("page1", "Targeted post", targeting, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "789" {
		t.Errorf("got id %q, want %q", id, "789")
	}
}

func TestPostText_WithSchedule(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertFormValue(t, r, "published", "false")
		testutil.AssertFormValue(t, r, "scheduled_publish_time", "2000000000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "sched_1"}`))
	})
	defer server.Close()

	id, err := client.PostText("page1", "Scheduled", "", 2000000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sched_1" {
		t.Errorf("got id %q, want %q", id, "sched_1")
	}
}

func TestPostText_Error(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Invalid message", "code": 100},
		})
	})
	defer server.Close()

	_, err := client.PostText("page1", "bad", "", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected status 400 in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// postLink tests
// ---------------------------------------------------------------------------

func TestPostLink_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/feed")
		testutil.AssertFormValue(t, r, "message", "Check this")
		testutil.AssertFormValue(t, r, "link", "https://example.com")
		testutil.AssertFormValue(t, r, "access_token", "tok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "link_1"}`))
	})
	defer server.Close()

	id, err := client.PostLink("page1", "Check this", "https://example.com", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "link_1" {
		t.Errorf("got id %q, want %q", id, "link_1")
	}
}

// ---------------------------------------------------------------------------
// postImageURL tests
// ---------------------------------------------------------------------------

func TestPostImageURL_Success(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")
		testutil.AssertFormValue(t, r, "url", "https://example.com/photo.jpg")
		testutil.AssertFormValue(t, r, "caption", "My photo")
		testutil.AssertFormValue(t, r, "access_token", "tok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_1"}`))
	})
	defer server.Close()

	id, err := client.PostImageURL("page1", "My photo", "https://example.com/photo.jpg", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "img_1" {
		t.Errorf("got id %q, want %q", id, "img_1")
	}
}

// ---------------------------------------------------------------------------
// postImageFile tests
// ---------------------------------------------------------------------------

func TestPostImageFile_Success(t *testing.T) {
	imgPath := createTempFile(t, "photo.jpg", "fake-image-data")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_123"}`))
	})
	defer server.Close()

	id, err := client.PostImageFile("page1", "Photo caption", imgPath, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "img_123" {
		t.Errorf("got id %q, want %q", id, "img_123")
	}
}

func TestPostImageFile_APIError(t *testing.T) {
	imgPath := createTempFile(t, "error.jpg", "will-fail")

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Invalid photo", "code": 100},
		})
	})
	defer server.Close()

	_, err := client.PostImageFile("page1", "Bad", imgPath, "", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected status 400 in error, got: %v", err)
	}
}

func TestPostImageFile_NetworkError(t *testing.T) {
	imgPath := createTempFile(t, "netfail.jpg", "will-timeout")

	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.PostImageFile("page1", "test", imgPath, "", 0)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestPostImageFile_FileNotFound(t *testing.T) {
	client := New(&http.Client{Timeout: time.Second}, "tok")
	_, err := client.PostImageFile("page1", "test", "/tmp/nonexistent_photo_xyz.jpg", "", 0)
	if err == nil {
		t.Fatal("expected file-not-found error, got nil")
	}
}

// ---------------------------------------------------------------------------
// uploadPhotoDraft tests
// ---------------------------------------------------------------------------

func TestUploadPhotoDraft_URL_Success(t *testing.T) {
	img := domain.ImageInput{Type: "url", Path: "https://example.com/pic.jpg", Filename: "pic.jpg"}

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")
		testutil.AssertFormValue(t, r, "url", "https://example.com/pic.jpg")
		testutil.AssertFormValue(t, r, "published", "false")
		testutil.AssertFormValue(t, r, "access_token", "tok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "draft_url_1"}`))
	})
	defer server.Close()

	id, err := client.UploadPhotoDraft("page1", img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "draft_url_1" {
		t.Errorf("got id %q, want %q", id, "draft_url_1")
	}
}

func TestUploadPhotoDraft_File_Success(t *testing.T) {
	imgPath := createTempFile(t, "draft.jpg", "draft-image-data")
	img := domain.ImageInput{Type: "file", Path: imgPath, Filename: "draft.jpg"}

	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "draft_file_1"}`))
	})
	defer server.Close()

	id, err := client.UploadPhotoDraft("page1", img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "draft_file_1" {
		t.Errorf("got id %q, want %q", id, "draft_file_1")
	}
}

func TestUploadPhotoDraft_File_NetworkError(t *testing.T) {
	imgPath := createTempFile(t, "netfail_draft.jpg", "will-timeout")
	img := domain.ImageInput{Type: "file", Path: imgPath, Filename: "netfail_draft.jpg"}

	client := New(&http.Client{Timeout: time.Millisecond}, "tok")
	_, err := client.UploadPhotoDraft("page1", img)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// ---------------------------------------------------------------------------
// handleResponse tests
// ---------------------------------------------------------------------------

func TestHandleResponse_SuccessWithID(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id": "abc_123"}`)),
	}
	id, err := HandleResponse(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "abc_123" {
		t.Errorf("got id %q, want %q", id, "abc_123")
	}
}

func TestHandleResponse_Error(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     "400 Bad Request",
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad request"}}`)),
	}
	_, err := HandleResponse(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected 400 in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newTestClient creates a facebook.Client backed by a test server.
func newTestClient(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(handler))
	httpClient := &http.Client{
		Transport: &testutil.RewriteTransport{Origin: "graph.facebook.com", Target: server.Listener.Addr().String()},
	}
	return New(httpClient, "tok"), server
}

// createTempFile creates a temporary file with the given content and returns its path.
func createTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create temp file %s: %v", path, err)
	}
	return path
}
