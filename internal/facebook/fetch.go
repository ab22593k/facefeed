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

// FetchAdsPosts fetches ads (unpublished) posts for a target and returns them.
func FetchAdsPosts(targetID, accessToken string, limit int) ([]AdsPost, error) {
	fields := "id,message,is_published,scheduled_publish_time,created_time,permalink_url"
	apiURL := GraphAPIURL(targetID+"/ads_posts") +
		fmt.Sprintf("?fields=%s&access_token=%s&limit=%d", fields, accessToken, limit)

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching unpublished posts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var fbErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &fbErr); err == nil && fbErr.Error.Message != "" {
			return nil, fmt.Errorf("Failed to fetch unpublished posts. Status: %s: %s", resp.Status, fbErr.Error.Message)
		}
		return nil, fmt.Errorf("Failed to fetch unpublished posts. Status: %s", resp.Status)
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

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching scheduled posts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var fbErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &fbErr); err == nil && fbErr.Error.Message != "" {
			return nil, fmt.Errorf("Failed to fetch scheduled posts. Status: %s: %s", resp.Status, fbErr.Error.Message)
		}
		return nil, fmt.Errorf("Failed to fetch scheduled posts. Status: %s", resp.Status)
	}

	var result scheduledPostsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Error parsing response: %w", err)
	}

	return result.Data, nil
}
