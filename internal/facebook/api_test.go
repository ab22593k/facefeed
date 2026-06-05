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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/feed")
		testutil.AssertFormValue(t, r, "message", "Hello, world!")
		testutil.AssertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "123_456"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := PostText("page1", "Hello, world!", "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "123_456" {
			t.Errorf("got id %q, want %q", id, "123_456")
		}
	})
}

func TestPostText_WithTargeting(t *testing.T) {
	targeting := `{"geo_locations":{"countries":["US"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertFormValue(t, r, "targeting", targeting)
		testutil.AssertFormValue(t, r, "message", "Targeted post")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "789"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := PostText("page1", "Targeted post", "tok", targeting, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "789" {
			t.Errorf("got id %q, want %q", id, "789")
		}
	})
}

func TestPostText_WithSchedule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertFormValue(t, r, "published", "false")
		testutil.AssertFormValue(t, r, "scheduled_publish_time", "2000000000")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "sched_1"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := PostText("page1", "Scheduled", "tok", "", 2000000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "sched_1" {
			t.Errorf("got id %q, want %q", id, "sched_1")
		}
	})
}

func TestPostText_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Invalid message", "code": 100},
		})
	}))
	defer server.Close()

	withFBClient(server, func() {
		_, err := PostText("page1", "bad", "tok", "", 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Errorf("expected status 400 in error, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// postLink tests
// ---------------------------------------------------------------------------

func TestPostLink_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/feed")
		testutil.AssertFormValue(t, r, "message", "Check this")
		testutil.AssertFormValue(t, r, "link", "https://example.com")
		testutil.AssertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "link_1"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := PostLink("page1", "Check this", "https://example.com", "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "link_1" {
			t.Errorf("got id %q, want %q", id, "link_1")
		}
	})
}

func TestPostLink_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Permission denied","code":200}}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		_, err := PostLink("page1", "test", "https://example.com", "tok", "", 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("expected status 403 in error, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// postImageURL tests
// ---------------------------------------------------------------------------

func TestPostImageURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")
		testutil.AssertFormValue(t, r, "url", "https://example.com/photo.jpg")
		testutil.AssertFormValue(t, r, "caption", "My photo")
		testutil.AssertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_1"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := PostImageURL("page1", "My photo", "https://example.com/photo.jpg", "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "img_1" {
			t.Errorf("got id %q, want %q", id, "img_1")
		}
	})
}

// ---------------------------------------------------------------------------
// postImageFile tests
// ---------------------------------------------------------------------------

func TestPostImageFile_Success(t *testing.T) {
	imgPath := createTempFile(t, "photo.jpg", "fake-image-data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_123"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := PostImageFile("page1", "Photo caption", imgPath, "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "img_123" {
			t.Errorf("got id %q, want %q", id, "img_123")
		}
	})
}

func TestPostImageFile_APIError(t *testing.T) {
	imgPath := createTempFile(t, "error.jpg", "will-fail")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Invalid photo", "code": 100},
		})
	}))
	defer server.Close()

	withFBClient(server, func() {
		_, err := PostImageFile("page1", "Bad", imgPath, "tok", "", 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Errorf("expected status 400 in error, got: %v", err)
		}
	})
}

func TestPostImageFile_NetworkError(t *testing.T) {
	imgPath := createTempFile(t, "netfail.jpg", "will-timeout")

	origClient := GetClient()
	SetClient(&http.Client{Timeout: time.Millisecond})
	defer SetClient(origClient)

	_, err := PostImageFile("page1", "test", imgPath, "tok", "", 0)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestPostImageFile_FileNotFound(t *testing.T) {
	_, err := PostImageFile("page1", "test", "/tmp/nonexistent_photo_xyz.jpg", "tok", "", 0)
	if err == nil {
		t.Fatal("expected file-not-found error, got nil")
	}
}

// ---------------------------------------------------------------------------
// uploadPhotoDraft tests
// ---------------------------------------------------------------------------

func TestUploadPhotoDraft_URL_Success(t *testing.T) {
	img := domain.ImageInput{Type: "url", Path: "https://example.com/pic.jpg", Filename: "pic.jpg"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")
		testutil.AssertFormValue(t, r, "url", "https://example.com/pic.jpg")
		testutil.AssertFormValue(t, r, "published", "false")
		testutil.AssertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "draft_url_1"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := UploadPhotoDraft("page1", img, "tok")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "draft_url_1" {
			t.Errorf("got id %q, want %q", id, "draft_url_1")
		}
	})
}

func TestUploadPhotoDraft_File_Success(t *testing.T) {
	imgPath := createTempFile(t, "draft.jpg", "draft-image-data")
	img := domain.ImageInput{Type: "file", Path: imgPath, Filename: "draft.jpg"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.AssertMethod(t, r, http.MethodPost)
		testutil.AssertPathSuffix(t, r, "/photos")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "draft_file_1"}`))
	}))
	defer server.Close()

	withFBClient(server, func() {
		id, err := UploadPhotoDraft("page1", img, "tok")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "draft_file_1" {
			t.Errorf("got id %q, want %q", id, "draft_file_1")
		}
	})
}

func TestUploadPhotoDraft_File_NetworkError(t *testing.T) {
	imgPath := createTempFile(t, "netfail_draft.jpg", "will-timeout")
	img := domain.ImageInput{Type: "file", Path: imgPath, Filename: "netfail_draft.jpg"}

	origClient := GetClient()
	SetClient(&http.Client{Timeout: time.Millisecond})
	defer SetClient(origClient)

	_, err := UploadPhotoDraft("page1", img, "tok")
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

func TestHandleResponse_SuccessNoID(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"success": true}`)),
	}
	id, err := HandleResponse(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != `{"success": true}` {
		t.Errorf("got %q, want raw body", id)
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

// withFBClient temporarily overrides the global client to point at a test
// server and restores it after fn completes.
func withFBClient(server *httptest.Server, fn func()) {
	orig := GetClient()
	SetClient(&http.Client{
		Transport: &testutil.RewriteTransport{Origin: "graph.facebook.com", Target: server.Listener.Addr().String()},
	})
	defer SetClient(orig)
	fn()
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
