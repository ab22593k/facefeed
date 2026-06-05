package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DeletePostByID deletes a Facebook post by its ID.
func (c *fbClient) DeletePostByID(postID string) error {
	deleteURL := GraphAPIURL(postID) + fmt.Sprintf("?access_token=%s", c.accessToken)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("error creating delete request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending delete request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	msg := fmt.Sprintf("failed to delete post. Status: %s", resp.Status)
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		msg = msg + ": " + errResp.Error.Message
	}
	return fmt.Errorf("%s", msg)
}
