package facebook

import (
	"net/http"
	"time"
)

// GraphAPIVersion is the Facebook Graph API version used for all API requests.
// Bump this to upgrade the API version across all endpoints.
const GraphAPIVersion = "v25.0"

// client is the shared HTTP client used across all Facebook API calls.
var client *http.Client

// InitClient sets up the HTTP client with the given timeout.
func InitClient(timeout time.Duration) {
	client = &http.Client{Timeout: timeout}
}

// GraphAPIURL returns the base URL for a Graph API resource.
func GraphAPIURL(path string) string {
	return "https://graph.facebook.com/" + GraphAPIVersion + "/" + path
}

// GetClient returns the current HTTP client (for use in tests that override it).
func GetClient() *http.Client {
	return client
}

// SetClient sets the HTTP client (for test overrides).
func SetClient(c *http.Client) {
	client = c
}
