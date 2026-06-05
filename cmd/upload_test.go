package cmd

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

// createTempFile creates a temporary file with the given content and returns
// its path. The file is removed when the test completes (t.TempDir cleanup).
func createTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create temp file %s: %v", path, err)
	}
	return path
}

// parseMultipartForm parses the request's multipart form and returns it.
// Use this once per handler, then read from r.MultipartForm directly.
func parseMultipartForm(t *testing.T, r *http.Request) *multipart.Form {
	t.Helper()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("failed to parse multipart form: %v", err)
	}
	return r.MultipartForm
}

// checkMultipartField asserts that a multipart form field has the expected value.
// form must come from a prior call to parseMultipartForm.
func checkMultipartField(t *testing.T, form *multipart.Form, key, want string) {
	t.Helper()
	vals := form.Value[key]
	if len(vals) != 1 {
		t.Errorf("multipart field %q: got %d values, want 1 (values: %v)", key, len(vals), vals)
		return
	}
	if vals[0] != want {
		t.Errorf("multipart field %q = %q, want %q", key, vals[0], want)
	}
}

// checkMultipartFile asserts that a multipart file upload field contains the
// expected content. form must come from a prior call to parseMultipartForm.
func checkMultipartFile(t *testing.T, form *multipart.Form, field, expectedContent string) {
	t.Helper()
	fhs := form.File[field]
	if len(fhs) == 0 {
		t.Errorf("multipart file field %q: no files uploaded", field)
		return
	}
	f, err := fhs[0].Open()
	if err != nil {
		t.Fatalf("failed to open uploaded file %s: %v", field, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read uploaded file %s: %v", field, err)
	}
	if string(data) != expectedContent {
		t.Errorf("uploaded file %q content = %q, want %q", field, string(data), expectedContent)
	}
}

// ---------------------------------------------------------------------------
// postImageFile tests
// ---------------------------------------------------------------------------

func TestPostImageFile_Success(t *testing.T) {
	imgPath := createTempFile(t, "photo.jpg", "fake-image-data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/photos")
		form := parseMultipartForm(t, r)
		checkMultipartField(t, form, "access_token", "tok")
		checkMultipartField(t, form, "caption", "Photo caption")
		checkMultipartFile(t, form, "source", "fake-image-data")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_123"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageFile("page1", "Photo caption", imgPath, "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "img_123" {
			t.Errorf("got id %q, want %q", id, "img_123")
		}
	})
}

func TestPostImageFile_WithTargeting(t *testing.T) {
	imgPath := createTempFile(t, "targeted.jpg", "targeted-content")
	targeting := `{"geo_locations":{"countries":["US"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form := parseMultipartForm(t, r)
		checkMultipartField(t, form, "targeting", targeting)
		checkMultipartField(t, form, "caption", "Targeted photo")
		checkMultipartFile(t, form, "source", "targeted-content")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "tgt_456"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageFile("page1", "Targeted photo", imgPath, "tok", targeting, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "tgt_456" {
			t.Errorf("got id %q, want %q", id, "tgt_456")
		}
	})
}

func TestPostImageFile_WithSchedule(t *testing.T) {
	imgPath := createTempFile(t, "scheduled.jpg", "scheduled-content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form := parseMultipartForm(t, r)
		checkMultipartField(t, form, "published", "false")
		checkMultipartField(t, form, "scheduled_publish_time", "2100000000")
		checkMultipartFile(t, form, "source", "scheduled-content")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "sched_789"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageFile("page1", "Scheduled photo", imgPath, "tok", "", 2100000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "sched_789" {
			t.Errorf("got id %q, want %q", id, "sched_789")
		}
	})
}

func TestPostImageFile_WithTargetingAndSchedule(t *testing.T) {
	imgPath := createTempFile(t, "full.jpg", "full-content")
	targeting := `{"geo_locations":{"countries":["GB"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		form := parseMultipartForm(t, r)
		checkMultipartField(t, form, "targeting", targeting)
		checkMultipartField(t, form, "published", "false")
		checkMultipartField(t, form, "scheduled_publish_time", "2200000000")
		checkMultipartFile(t, form, "source", "full-content")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "full_000"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageFile("page1", "Full photo", imgPath, "tok", targeting, 2200000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "full_000" {
			t.Errorf("got id %q, want %q", id, "full_000")
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

	withTestClient(server, func() {
		_, err := postImageFile("page1", "Bad", imgPath, "tok", "", 0)
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

	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	_, err := postImageFile("page1", "test", imgPath, "tok", "", 0)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestPostImageFile_FileNotFound(t *testing.T) {
	_, err := postImageFile("page1", "test", "/tmp/nonexistent_photo_xyz.jpg", "tok", "", 0)
	if err == nil {
		t.Fatal("expected file-not-found error, got nil")
	}
}

// ---------------------------------------------------------------------------
// uploadPhotoDraft tests – URL mode
// ---------------------------------------------------------------------------

func TestUploadPhotoDraft_URL_Success(t *testing.T) {
	img := ImageInput{Type: "url", Path: "https://example.com/pic.jpg", Filename: "pic.jpg"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/photos")
		assertFormValue(t, r, "url", "https://example.com/pic.jpg")
		assertFormValue(t, r, "published", "false")
		assertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "draft_url_1"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := uploadPhotoDraft("page1", img, "tok")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "draft_url_1" {
			t.Errorf("got id %q, want %q", id, "draft_url_1")
		}
	})
}

func TestUploadPhotoDraft_URL_APIError(t *testing.T) {
	img := ImageInput{Type: "url", Path: "https://example.com/bad.jpg", Filename: "bad.jpg"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Permission denied","code":200}}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		_, err := uploadPhotoDraft("page1", img, "tok")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("expected status 403 in error, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// uploadPhotoDraft tests – file mode (multipart)
// ---------------------------------------------------------------------------

func TestUploadPhotoDraft_File_Success(t *testing.T) {
	imgPath := createTempFile(t, "draft.jpg", "draft-image-data")
	img := ImageInput{Type: "file", Path: imgPath, Filename: "draft.jpg"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/photos")
		form := parseMultipartForm(t, r)
		checkMultipartField(t, form, "access_token", "tok")
		checkMultipartField(t, form, "published", "false")
		checkMultipartFile(t, form, "source", "draft-image-data")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "draft_file_1"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := uploadPhotoDraft("page1", img, "tok")
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
	img := ImageInput{Type: "file", Path: imgPath, Filename: "netfail_draft.jpg"}

	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	_, err := uploadPhotoDraft("page1", img, "tok")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestUploadPhotoDraft_File_NotFound(t *testing.T) {
	img := ImageInput{Type: "file", Path: "/tmp/missing_draft_xyz.jpg", Filename: "missing.jpg"}

	_, err := uploadPhotoDraft("page1", img, "tok")
	if err == nil {
		t.Fatal("expected file-not-found error, got nil")
	}
}
