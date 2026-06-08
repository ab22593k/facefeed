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
func checkFBStatus(body []byte, resp *http.Response, prefix string) error {
	var fbErr facebookError
	if err := json.Unmarshal(body, &fbErr); err == nil && fbErr.Error.Message != "" {
		return fmt.Errorf("%s. Status: %s: %s", prefix, resp.Status, fbErr.Error.Message)
	}
	return fmt.Errorf("%s. Status: %s", prefix, resp.Status)
}

// doFBGet performs a GET request using the client's httpClient.
func (c *fbClient) doFBGet(apiURL string) ([]byte, error) {
	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, checkFBStatus(body, resp, "Failed to fetch")
	}
	return body, nil
}

// FetchAdsPosts fetches ads (unpublished) posts for a target and returns them.
func (c *fbClient) FetchAdsPosts(targetID string, limit int) ([]AdsPost, error) {
	fields := "id,message,is_published,scheduled_publish_time,created_time,permalink_url"
	apiURL := GraphAPIURL(targetID+"/ads_posts") +
		fmt.Sprintf("?fields=%s&access_token=%s&limit=%d", fields, c.accessToken, limit)

	body, err := c.doFBGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching unpublished posts: %w", err)
	}

	var result adsPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return result.Data, nil
}

// FetchScheduledPosts fetches scheduled posts for a target and returns them.
func (c *fbClient) FetchScheduledPosts(targetID string, limit int) ([]ScheduledPost, error) {
	apiURL := GraphAPIURL(targetID+"/scheduled_posts") +
		fmt.Sprintf("?access_token=%s&limit=%d", c.accessToken, limit)

	body, err := c.doFBGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching scheduled posts: %w", err)
	}

	var result scheduledPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return result.Data, nil
}

// FetchLatestFeedPost fetches the most recent post ID for a target's feed.
func (c *fbClient) FetchLatestFeedPost(targetID string) (string, error) {
	apiURL := GraphAPIURL(targetID+"/feed") +
		fmt.Sprintf("?limit=1&access_token=%s", c.accessToken)

	body, err := c.doFBGet(apiURL)
	if err != nil {
		return "", fmt.Errorf("error fetching latest feed post: %w", err)
	}

	var feed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &feed); err != nil {
		return "", fmt.Errorf("error parsing feed response: %w", err)
	}

	if len(feed.Data) == 0 {
		return "", fmt.Errorf("no posts found for feed %s", targetID)
	}

	return feed.Data[0].ID, nil
}
