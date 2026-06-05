package testutil

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"
)

// RewriteTransport is an http.RoundTripper that rewrites the host in requests
// so they can be directed to a test server while preserving the original URL path.
type RewriteTransport struct {
	Origin string
	Target string
}

func (rt *RewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Host = rt.Target
	clone.URL.Scheme = "http"
	clone.Host = rt.Target
	return http.DefaultTransport.RoundTrip(clone)
}

// NewRewriteTransport creates a transport that rewrites origin host to target address.
func NewRewriteTransport(origin, target string) *RewriteTransport {
	return &RewriteTransport{Origin: origin, Target: target}
}

// CaptureStdout executes fn and returns everything written to stdout as a string.
func CaptureStdout(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic("failed to create pipe: " + err.Error())
	}

	original := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	os.Stdout = original

	return buf.String()
}

// AssertMethod checks the HTTP method of a request.
func AssertMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Errorf("expected method %s, got %s", want, r.Method)
	}
}

// AssertPathSuffix checks that a request path ends with the expected suffix.
func AssertPathSuffix(t *testing.T, r *http.Request, suffix string) {
	t.Helper()
	if !stringsHasSuffix(r.URL.Path, suffix) {
		t.Errorf("expected path to end with %s, got %s", suffix, r.URL.Path)
	}
}

// AssertFormValue checks that a parsed form value matches the expected value.
func AssertFormValue(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("failed to parse form: %v", err)
	}
	got := r.Form.Get(key)
	if got != want {
		t.Errorf("form[%q] = %q, want %q", key, got, want)
	}
}

// stringsHasSuffix is a helper for AssertPathSuffix.
func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
