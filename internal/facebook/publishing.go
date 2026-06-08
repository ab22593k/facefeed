package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"facefeed/domain"
)

// PostVideoUpload uploads a video file and publishes it to a Page feed.
func (c *fbClient) PostVideoUpload(pageID, title, description, filePath string, scheduleUnix int64) (string, error) {
	apiURL := "https://graph-video.facebook.com/" + GraphAPIVersion + "/" + pageID + "/videos"

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open video file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)

	errChan := make(chan error, 1)

	go func() {
		defer func() { _ = bodyWriter.Close() }()
		defer func() { _ = multiWriter.Close() }()

		_ = multiWriter.WriteField("access_token", c.accessToken)
		if title != "" {
			_ = multiWriter.WriteField("title", title)
		}
		if description != "" {
			_ = multiWriter.WriteField("description", description)
		}
		if scheduleUnix > 0 {
			_ = multiWriter.WriteField("published", "false")
			_ = multiWriter.WriteField("scheduled_publish_time", fmt.Sprintf("%d", scheduleUnix))
		}

		part, err := multiWriter.CreateFormFile("source", filepath.Base(filePath))
		if err != nil {
			errChan <- err
			return
		}

		_, err = io.Copy(part, file)
		if err != nil {
			errChan <- err
			return
		}
	}()

	req, err := http.NewRequest("POST", apiURL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create video upload request: %w", err)
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload video: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	select {
	case err := <-errChan:
		return "", err
	default:
		return HandleResponse(resp)
	}
}

// insightResponse wraps the API response for the /insights endpoint.
type insightResponse struct {
	Data   []domain.InsightData `json:"data"`
	Paging *struct {
		Next string `json:"next"`
	} `json:"paging"`
}

// GetInsights retrieves insight metrics for a Page or Post.
func (c *fbClient) GetInsights(objectID string, metric string, period string) ([]domain.InsightData, error) {
	apiURL := GraphAPIURL(objectID+"/insights") +
		fmt.Sprintf("?metric=%s&access_token=%s&period=%s", url.QueryEscape(metric), c.accessToken, url.QueryEscape(period))

	body, err := c.doFBGet(apiURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching insights: %w", err)
	}

	var result insightResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing insights response: %w", err)
	}

	return result.Data, nil
}

// startUploadResponse wraps the response from Reels/Stories upload start phase.
type startUploadResponse struct {
	VideoID   string `json:"video_id"`
	UploadURL string `json:"upload_url"`
}

// uploadFileToURL uploads a file to a given URL via POST.
func (c *fbClient) uploadFileToURL(uploadURL, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	req, err := http.NewRequest("POST", uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Content-Type", "video/mp4")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// doStartUploadPhase sends the start phase of a Reels/Stories upload and returns the response.
func (c *fbClient) doStartUploadPhase(apiURL, filePath string) (startUploadResponse, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return startUploadResponse{}, fmt.Errorf("failed to stat file: %w", err)
	}

	data := url.Values{
		"access_token": {c.accessToken},
		"upload_phase": {"start"},
		"file_size":    {fmt.Sprintf("%d", info.Size())},
	}

	resp, err := c.httpClient.PostForm(apiURL, data)
	if err != nil {
		return startUploadResponse{}, fmt.Errorf("failed to start upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return startUploadResponse{}, fmt.Errorf("failed to read start response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return startUploadResponse{}, checkFBStatus(body, resp, "Upload start failed")
	}

	var startResp startUploadResponse
	if err := json.Unmarshal(body, &startResp); err != nil {
		return startUploadResponse{}, fmt.Errorf("failed to parse start response: %w", err)
	}
	if startResp.VideoID == "" {
		return startUploadResponse{}, fmt.Errorf("start response missing video_id: %s", string(body))
	}

	return startResp, nil
}

// doFinishUploadPhase sends the finish phase of a Reels/Stories upload.
func (c *fbClient) doFinishUploadPhase(apiURL, videoID string, extra url.Values) ([]byte, error) {
	data := url.Values{
		"access_token": {c.accessToken},
		"upload_phase": {"finish"},
		"video_id":     {videoID},
	}
	for k, vals := range extra {
		data[k] = vals
	}

	resp, err := c.httpClient.PostForm(apiURL, data)
	if err != nil {
		return nil, fmt.Errorf("failed to finish upload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read finish response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, checkFBStatus(body, resp, "Upload finish failed")
	}

	return body, nil
}

// PublishReel uploads a video as a Facebook Reel using the two-phase upload API.
func (c *fbClient) PublishReel(pageID, description, filePath string) (string, error) {
	apiURL := GraphAPIURL(pageID + "/video_reels")

	// Phase 1: Start
	startResp, err := c.doStartUploadPhase(apiURL, filePath)
	if err != nil {
		return "", err
	}

	// Phase 2: Upload file to the upload URL
	if err := c.uploadFileToURL(startResp.UploadURL, filePath); err != nil {
		return "", err
	}

	// Phase 3: Finish
	extra := url.Values{
		"video_state": {"PUBLISHED"},
	}
	if description != "" {
		extra["description"] = []string{description}
	}

	_, err = c.doFinishUploadPhase(apiURL, startResp.VideoID, extra)
	if err != nil {
		return "", err
	}

	return startResp.VideoID, nil
}

// PublishStory publishes a video or photo as a Facebook Story.
func (c *fbClient) PublishStory(pageID, filePath string, isVideo bool) (string, error) {
	if isVideo {
		return c.publishVideoStory(pageID, filePath)
	}
	return c.publishPhotoStory(pageID, filePath)
}

// publishVideoStory handles the two-phase video story upload.
func (c *fbClient) publishVideoStory(pageID, filePath string) (string, error) {
	apiURL := GraphAPIURL(pageID + "/video_stories")

	// Phase 1: Start
	startResp, err := c.doStartUploadPhase(apiURL, filePath)
	if err != nil {
		return "", err
	}

	// Phase 2: Upload file
	if err := c.uploadFileToURL(startResp.UploadURL, filePath); err != nil {
		return "", err
	}

	// Phase 3: Finish
	body, err := c.doFinishUploadPhase(apiURL, startResp.VideoID, nil)
	if err != nil {
		return "", err
	}

	var finishResult struct {
		Success bool   `json:"success"`
		PostID  string `json:"post_id"`
	}
	if err := json.Unmarshal(body, &finishResult); err == nil && finishResult.PostID != "" {
		return finishResult.PostID, nil
	}

	return startResp.VideoID, nil
}

// publishPhotoStory uploads a photo as a Facebook Story.
// This requires two steps: upload photo as a draft, then create the story with the photo_id.
func (c *fbClient) publishPhotoStory(pageID, filePath string) (string, error) {
	// Step 1: Upload photo as draft to get a photo_id
	img := domain.ImageInput{
		Path:     filePath,
		Type:     "file",
		Filename: filepath.Base(filePath),
	}

	photoID, err := c.UploadPhotoDraft(pageID, img)
	if err != nil {
		return "", fmt.Errorf("failed to upload photo draft for story: %w", err)
	}

	// Step 2: Create photo story with the photo_id
	apiURL := GraphAPIURL(pageID + "/photo_stories")
	data := url.Values{
		"access_token": {c.accessToken},
		"photo_id":     {photoID},
	}

	resp, err := c.httpClient.PostForm(apiURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to publish photo story: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read photo story response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", checkFBStatus(body, resp, "Photo story publish failed")
	}

	// Try to extract post_id from response
	var res struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
		PostID  string `json:"post_id"`
	}
	if err := json.Unmarshal(body, &res); err == nil {
		if res.PostID != "" {
			return res.PostID, nil
		}
		if res.ID != "" {
			return res.ID, nil
		}
	}

	return photoID, nil
}

// ReplyToComment posts a reply to an existing comment.
func (c *fbClient) ReplyToComment(commentID, message string) (string, error) {
	apiURL := GraphAPIURL(commentID + "/comments")
	data := map[string]string{
		"access_token": c.accessToken,
		"message":      message,
	}
	return c.postForm(apiURL, data)
}

// UpdatePost updates the message/content of an existing post.
func (c *fbClient) UpdatePost(postID, message string) error {
	apiURL := GraphAPIURL(postID)
	form := url.Values{
		"access_token": {c.accessToken},
		"message":      {message},
	}

	resp, err := c.httpClient.PostForm(apiURL, form)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read update response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("failed to update post. Status: %s", resp.Status)
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			msg = msg + ": " + errResp.Error.Message
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
}
