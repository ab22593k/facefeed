package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	theme "facefeed/internal"
)

// DeletePostByID deletes a Facebook post by its ID.
func DeletePostByID(postID, accessToken string) {
	theme.PrintSection(fmt.Sprintf("Delete Post %s", postID))

	theme.Info("Post ID", postID)
	theme.Info("Status", "Deleting...")

	deleteURL := GraphAPIURL(postID) + fmt.Sprintf("?access_token=%s", accessToken)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		theme.Error(fmt.Sprintf("Error creating delete request: %v", err))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		theme.Error(fmt.Sprintf("Error sending delete request: %v", err))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		theme.Success(fmt.Sprintf("Successfully deleted post: %s", postID))
	} else {
		theme.Error(fmt.Sprintf("Failed to delete post. Status: %s", resp.Status))
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			theme.Error(errResp.Error.Message)
		}
	}
}
