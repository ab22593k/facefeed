package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AdsPost represents a single unpublished post from the ads_posts API response.
type AdsPost struct {
	ID                   string `json:"id"`
	Message              string `json:"message"`
	IsPublished          bool   `json:"is_published"`
	ScheduledPublishTime int64  `json:"scheduled_publish_time"`
	CreatedTime          string `json:"created_time"`
	PermalinkURL         string `json:"permalink_url"`
}

// adsPostsResponse wraps the paginated API response for ads_posts.
type adsPostsResponse struct {
	Data   []AdsPost `json:"data"`
	Paging *struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// ScheduledPost represents a single scheduled post from the API response.
type ScheduledPost struct {
	ID                   string `json:"id"`
	Message              string `json:"message"`
	ScheduledPublishTime int64  `json:"scheduled_publish_time"`
	CreatedTime          string `json:"created_time"`
	Status               *struct {
		Published bool `json:"published"`
		Scheduled bool `json:"scheduled"`
	} `json:"status"`
}

// scheduledPostsResponse wraps the paginated API response for scheduled_posts.
type scheduledPostsResponse struct {
	Data   []ScheduledPost `json:"data"`
	Paging *struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// facebookError wraps the FB API error envelope.
type facebookError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// checkFBStatus parses a non-200 Facebook API response and returns a descriptive error.
// If the response body contains a Facebook error message, it is included in the result.
func checkFBStatus(body []byte, resp *http.Response, prefix string) error {
	var fbErr facebookError
	if err := json.Unmarshal(body, &fbErr); err == nil && fbErr.Error.Message != "" {
		return fmt.Errorf("%s. Status: %s: %s", prefix, resp.Status, fbErr.Error.Message)
	}
	return fmt.Errorf("%s. Status: %s", prefix, resp.Status)
}

// doFBGet performs a GET request to the Facebook Graph API and reads the response body.
// It returns the raw body bytes and any transport-level error.
func doFBGet(apiURL string) ([]byte, error) {
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, checkFBStatus(body, resp, "Failed to fetch")
	}
	return body, nil
}

// FetchAdsPosts fetches ads (unpublished) posts for a target and returns them.
func FetchAdsPosts(targetID, accessToken string, limit int) ([]AdsPost, error) {
	fields := "id,message,is_published,scheduled_publish_time,created_time,permalink_url"
	apiURL := GraphAPIURL(targetID+"/ads_posts") +
		fmt.Sprintf("?fields=%s&access_token=%s&limit=%d", fields, accessToken, limit)

	body, err := doFBGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching unpublished posts: %w", err)
	}

	var result adsPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Error parsing response: %w", err)
	}

	return result.Data, nil
}

// FetchScheduledPosts fetches scheduled posts for a target and returns them.
func FetchScheduledPosts(targetID, accessToken string, limit int) ([]ScheduledPost, error) {
	apiURL := GraphAPIURL(targetID+"/scheduled_posts") +
		fmt.Sprintf("?access_token=%s&limit=%d", accessToken, limit)

	body, err := doFBGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching scheduled posts: %w", err)
	}

	var result scheduledPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Error parsing response: %w", err)
	}

	return result.Data, nil
}
