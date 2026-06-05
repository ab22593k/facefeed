package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// postText tests
// ---------------------------------------------------------------------------

func TestPostText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/feed")
		assertFormValue(t, r, "message", "Hello, world!")
		assertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "123_456"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postText("page1", "Hello, world!", "tok", "", 0)
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
		assertFormValue(t, r, "targeting", targeting)
		assertFormValue(t, r, "message", "Targeted post")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "789"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postText("page1", "Targeted post", "tok", targeting, 0)
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
		assertFormValue(t, r, "published", "false")
		assertFormValue(t, r, "scheduled_publish_time", "2000000000")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "sched_1"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postText("page1", "Scheduled", "tok", "", 2000000000)
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

	withTestClient(server, func() {
		_, err := postText("page1", "bad", "tok", "", 0)
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
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/feed")
		assertFormValue(t, r, "message", "Check this")
		assertFormValue(t, r, "link", "https://example.com")
		assertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "link_1"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postLink("page1", "Check this", "https://example.com", "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "link_1" {
			t.Errorf("got id %q, want %q", id, "link_1")
		}
	})
}

func TestPostLink_WithTargetingAndSchedule(t *testing.T) {
	targeting := `{"geo_locations":{"countries":["CA"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertFormValue(t, r, "targeting", targeting)
		assertFormValue(t, r, "published", "false")
		assertFormValue(t, r, "scheduled_publish_time", "2100000000")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "link_sched"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postLink("page1", "Linked", "https://example.com", "tok", targeting, 2100000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "link_sched" {
			t.Errorf("got id %q, want %q", id, "link_sched")
		}
	})
}

func TestPostLink_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Permission denied","code":200}}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		_, err := postLink("page1", "test", "https://example.com", "tok", "", 0)
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
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/photos")
		assertFormValue(t, r, "url", "https://example.com/photo.jpg")
		assertFormValue(t, r, "caption", "My photo")
		assertFormValue(t, r, "access_token", "tok")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_1"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageURL("page1", "My photo", "https://example.com/photo.jpg", "tok", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "img_1" {
			t.Errorf("got id %q, want %q", id, "img_1")
		}
	})
}

func TestPostImageURL_WithSchedule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertFormValue(t, r, "published", "false")
		assertFormValue(t, r, "scheduled_publish_time", "2200000000")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_sched"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageURL("page1", "Scheduled photo", "https://example.com/pic.jpg", "tok", "", 2200000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "img_sched" {
			t.Errorf("got id %q, want %q", id, "img_sched")
		}
	})
}

func TestPostImageURL_WithTargetingAndSchedule(t *testing.T) {
	targeting := `{"geo_locations":{"countries":["GB"]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertFormValue(t, r, "targeting", targeting)
		assertFormValue(t, r, "published", "false")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "img_full"}`))
	}))
	defer server.Close()

	withTestClient(server, func() {
		id, err := postImageURL("page1", "Full", "https://example.com/pic.jpg", "tok", targeting, 2300000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "img_full" {
			t.Errorf("got id %q, want %q", id, "img_full")
		}
	})
}

func TestPostImageURL_NetworkError(t *testing.T) {
	originalClient := client
	client = &http.Client{Timeout: time.Millisecond}
	defer func() { client = originalClient }()

	_, err := postImageURL("page1", "test", "http://example.com/photo.jpg", "tok", "", 0)
	// Note: the request is sent to graph.facebook.com (constructed internally),
	// which should reliably timeout at 1ms.
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
	id, err := handleResponse(resp)
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
	id, err := handleResponse(resp)
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
	_, err := handleResponse(resp)
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

// withTestClient temporarily overrides the global client to point at a test
// server and restores it after fn completes.
func withTestClient(server *httptest.Server, fn func()) {
	originalClient := client
	client = &http.Client{
		Transport: &rewriteTransport{origin: "graph.facebook.com", target: server.Listener.Addr().String()},
	}
	defer func() { client = originalClient }()
	fn()
}

// assertMethod checks the HTTP method of a request.
func assertMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Errorf("expected method %s, got %s", want, r.Method)
	}
}

// assertPathSuffix checks that a request path ends with the expected suffix.
func assertPathSuffix(t *testing.T, r *http.Request, suffix string) {
	t.Helper()
	if !strings.HasSuffix(r.URL.Path, suffix) {
		t.Errorf("expected path to end with %s, got %s", suffix, r.URL.Path)
	}
}

// assertFormValue checks that a parsed form value matches the expected value.
func assertFormValue(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("failed to parse form: %v", err)
	}
	got := r.Form.Get(key)
	if got != want {
		t.Errorf("form[%q] = %q, want %q", key, got, want)
	}
}
