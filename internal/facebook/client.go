package facebook

import (
	"net/http"
	"time"

	"facefeed/domain"
)

// GraphAPIVersion is the Facebook Graph API version used for all API requests.
const GraphAPIVersion = "v25.0"

// Client defines the Facebook Graph API operations available to commands.
// This is the abstraction that commands depend on instead of net/http directly.
type Client interface {
	PostText(targetID, message, targetingJSON string, scheduleUnix int64) (string, error)
	PostLink(targetID, message, link, targetingJSON string, scheduleUnix int64) (string, error)
	PostMultiPhoto(targetID, message string, mediaIDs []string, targetingJSON string, scheduleUnix int64) (string, error)
	PostImageURL(pageID, message, imageURL, targetingJSON string, scheduleUnix int64) (string, error)
	PostImageFile(pageID, message, filePath, targetingJSON string, scheduleUnix int64) (string, error)
	UploadPhotoDraft(pageID string, img domain.ImageInput) (string, error)
	PublishMultiPhoto(images []domain.ImageInput, message, pageID, targetingJSON string, scheduleUnix int64) (string, error)
	UploadMultipleImages(images []domain.ImageInput, message, pageID, targetingJSON string, scheduleUnix int64) []domain.UploadResult
	FetchAdsPosts(targetID string, limit int) ([]AdsPost, error)
	FetchScheduledPosts(targetID string, limit int) ([]ScheduledPost, error)
	FetchLatestFeedPost(targetID string) (string, error)
	DeletePostByID(postID string) error
}

// client is the concrete implementation of Client.
type fbClient struct {
	httpClient  *http.Client
	accessToken string
}

// New creates a new Facebook client with the given HTTP client and access token.
func New(httpClient *http.Client, accessToken string) Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &fbClient{
		httpClient:  httpClient,
		accessToken: accessToken,
	}
}

// GraphAPIURL returns the base URL for a Graph API resource.
func GraphAPIURL(path string) string {
	return "https://graph.facebook.com/" + GraphAPIVersion + "/" + path
}
