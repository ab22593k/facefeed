package facebook

import (
	"encoding/json"
	"net/url"
)

// PostText publishes a text-only post to a Facebook page or group.
func PostText(targetID, message, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(targetID + "/feed")
	data := map[string]string{
		"access_token": accessToken,
		"message":      message,
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return postForm(apiURL, data)
}

// PostLink publishes a link post to a Facebook page or group.
func PostLink(targetID, message, link, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(targetID + "/feed")
	data := map[string]string{
		"access_token": accessToken,
		"message":      message,
		"link":         link,
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return postForm(apiURL, data)
}

// PostMultiPhoto publishes a post with multiple photos (grid layout) using
// previously uploaded draft image IDs attached via attached_media.
func PostMultiPhoto(targetID, message string, mediaIDs []string, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(targetID + "/feed")

	attachedMedia := make([]map[string]string, len(mediaIDs))
	for i, id := range mediaIDs {
		attachedMedia[i] = map[string]string{"media_fbid": id}
	}

	attachedJSON, _ := json.Marshal(attachedMedia)

	data := map[string]string{
		"access_token":   accessToken,
		"message":        message,
		"attached_media": string(attachedJSON),
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return postForm(apiURL, data)
}

// postForm sends a POST request with form-encoded data and returns the response ID.
func postForm(apiURL string, data map[string]string) (string, error) {
	form := url.Values{}
	for k, v := range data {
		form[k] = []string{v}
	}

	resp, err := client.PostForm(apiURL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return HandleResponse(resp)
}
