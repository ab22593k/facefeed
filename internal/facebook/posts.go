package facebook

import (
	"encoding/json"
	"net/url"
)

// PostText publishes a text-only post to a Facebook page or group.
func (c *fbClient) PostText(targetID, message, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(targetID + "/feed")
	data := map[string]string{
		"access_token": c.accessToken,
		"message":      message,
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return c.postForm(apiURL, data)
}

// PostLink publishes a link post to a Facebook page or group.
func (c *fbClient) PostLink(targetID, message, link, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(targetID + "/feed")
	data := map[string]string{
		"access_token": c.accessToken,
		"message":      message,
		"link":         link,
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return c.postForm(apiURL, data)
}

// PostMultiPhoto publishes a post with multiple photos (grid layout) using
// previously uploaded draft image IDs attached via attached_media.
func (c *fbClient) PostMultiPhoto(targetID, message string, mediaIDs []string, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(targetID + "/feed")

	attachedMedia := make([]map[string]string, len(mediaIDs))
	for i, id := range mediaIDs {
		attachedMedia[i] = map[string]string{"media_fbid": id}
	}

	attachedJSON, _ := json.Marshal(attachedMedia)

	data := map[string]string{
		"access_token":   c.accessToken,
		"message":        message,
		"attached_media": string(attachedJSON),
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return c.postForm(apiURL, data)
}

// postForm sends a POST request with form-encoded data and returns the response ID.
func (c *fbClient) postForm(apiURL string, data map[string]string) (string, error) {
	form := url.Values{}
	for k, v := range data {
		form[k] = []string{v}
	}

	resp, err := c.httpClient.PostForm(apiURL, form)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	return HandleResponse(resp)
}
