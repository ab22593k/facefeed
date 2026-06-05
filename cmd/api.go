package cmd

import (
	theme "bubble/internal"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// addScheduleParams adds published=false and scheduled_publish_time to url.Values
// when a schedule time is specified.
func addScheduleParams(data url.Values, scheduleUnix int64) {
	if scheduleUnix > 0 {
		data["published"] = []string{"false"}
		data["scheduled_publish_time"] = []string{strconv.FormatInt(scheduleUnix, 10)}
	}
}

func uploadMultipleImages(images []ImageInput, message, accessToken, pageID, targetingJSON string, scheduleUnix int64) []UploadResult {
	var results []UploadResult

	if scheduleUnix > 0 {
		theme.Info("Schedule", fmt.Sprintf("Scheduled for %s", time.Unix(scheduleUnix, 0).Format(time.RFC3339)))
	}
	theme.PrintSection(fmt.Sprintf("Uploading %d Images", len(images)))

	for i, img := range images {
		fmt.Printf("\n%s[%d/%d]%s %s%s%s\n", theme.Blue, i+1, len(images), theme.Reset, theme.Bold, img.Filename, theme.Reset)

		var postID string
		var err error

		if img.Type == "url" {
			postID, err = postImageURL(pageID, message, img.Path, accessToken, targetingJSON, scheduleUnix)
		} else {
			postID, err = postImageFile(pageID, message, img.Path, accessToken, targetingJSON, scheduleUnix)
		}

		results = append(results, UploadResult{
			Filename: img.Filename,
			Success:  err == nil,
			PostID:   postID,
			Error:    err,
		})

		if i < len(images)-1 {
			theme.Info("Status", "Waiting 1s to avoid rate limits...")
			time.Sleep(1 * time.Second)
		}
	}

	return results
}

func postImageURL(pageID, message, imageURL, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/photos", pageID)
	data := url.Values{
		"access_token": {accessToken},
		"url":          {imageURL},
		"caption":      {message},
	}
	if targetingJSON != "" {
		data["targeting"] = []string{targetingJSON}
	}
	addScheduleParams(data, scheduleUnix)

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return handleResponse(resp)
}

func postImageFile(pageID, message, filePath, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/photos", pageID)

	uploadPath := filePath
	isSVG := strings.ToLower(filepath.Ext(filePath)) == ".svg"

	if isSVG {
		if _, err := exec.LookPath("convert"); err != nil {
			return "", fmt.Errorf("SVG files require ImageMagick (convert command not found).\nInstall: apt-get install imagemagick (Linux) or brew install imagemagick (macOS)")
		}

		tempPNG := filePath + ".temp.png"
		cmd := exec.Command("convert", filePath, tempPNG)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to convert SVG to PNG: %v", err)
		}
		uploadPath = tempPNG
		defer os.Remove(tempPNG)
	}

	file, err := os.Open(uploadPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, _ := file.Stat()

	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)

	errChan := make(chan error, 1)

	go func() {
		defer bodyWriter.Close()
		defer multiWriter.Close()

		_ = multiWriter.WriteField("access_token", accessToken)
		_ = multiWriter.WriteField("caption", message)
		if targetingJSON != "" {
			_ = multiWriter.WriteField("targeting", targetingJSON)
		}
		if scheduleUnix > 0 {
			_ = multiWriter.WriteField("published", "false")
			_ = multiWriter.WriteField("scheduled_publish_time", strconv.FormatInt(scheduleUnix, 10))
		}

		part, err := multiWriter.CreateFormFile("source", filepath.Base(uploadPath))
		if err != nil {
			errChan <- err
			return
		}

		bar := theme.NewProgressBar(info.Size(), "uploading")

		_, err = io.Copy(io.MultiWriter(part, bar), file)
		if err != nil {
			errChan <- err
			return
		}
	}()

	req, err := http.NewRequest("POST", apiURL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	select {
	case err := <-errChan:
		return "", err
	default:
		return handleResponse(resp)
	}
}

func postLink(targetID, message, link, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/feed", targetID)
	data := url.Values{
		"access_token": {accessToken},
		"message":      {message},
		"link":         {link},
	}
	if targetingJSON != "" {
		data["targeting"] = []string{targetingJSON}
	}
	addScheduleParams(data, scheduleUnix)

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return handleResponse(resp)
}

func postText(targetID, message, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/feed", targetID)
	data := url.Values{
		"access_token": {accessToken},
		"message":      {message},
	}
	if targetingJSON != "" {
		data["targeting"] = []string{targetingJSON}
	}
	addScheduleParams(data, scheduleUnix)

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return handleResponse(resp)
}

// uploadPhotoDraft uploads a single image as an unpublished draft and returns its media ID.
// It handles both URL and file-based images, including SVG-to-PNG conversion.
func uploadPhotoDraft(pageID string, img ImageInput, accessToken string) (string, error) {
	if img.Type == "url" {
		apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/photos", pageID)
		data := url.Values{
			"access_token": {accessToken},
			"url":          {img.Path},
			"published":    {"false"},
		}

		resp, err := client.PostForm(apiURL, data)
		if err != nil {
			return "", fmt.Errorf("failed to upload %s: %w", img.Filename, err)
		}
		defer resp.Body.Close()

		return handleResponse(resp)
	}

	// File-based upload (multipart)
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/photos", pageID)

	uploadPath := img.Path
	isSVG := strings.ToLower(filepath.Ext(img.Path)) == ".svg"

	if isSVG {
		if _, err := exec.LookPath("convert"); err != nil {
			return "", fmt.Errorf("SVG files require ImageMagick (convert command not found).\nInstall: apt-get install imagemagick (Linux) or brew install imagemagick (macOS)")
		}

		tempPNG := img.Path + ".temp.png"
		cmd := exec.Command("convert", img.Path, tempPNG)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to convert SVG to PNG: %v", err)
		}
		uploadPath = tempPNG
		defer os.Remove(tempPNG)
	}

	file, err := os.Open(uploadPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)

	errChan := make(chan error, 1)
	var mediaID string

	go func() {
		defer bodyWriter.Close()
		defer multiWriter.Close()

		_ = multiWriter.WriteField("access_token", accessToken)
		_ = multiWriter.WriteField("published", "false")

		part, err := multiWriter.CreateFormFile("source", filepath.Base(uploadPath))
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
		return "", err
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	select {
	case err := <-errChan:
		return "", err
	default:
		mediaID, err = handleResponse(resp)
		if err != nil {
			return "", fmt.Errorf("failed to upload draft %s: %w", img.Filename, err)
		}
		return mediaID, nil
	}
}

// postMultiPhoto publishes a post with multiple photos (grid layout) using
// previously uploaded draft image IDs attached via attached_media.
func postMultiPhoto(targetID, message string, mediaIDs []string, accessToken, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/"+graphAPIVersion+"/%s/feed", targetID)

	attachedMedia := make([]map[string]string, len(mediaIDs))
	for i, id := range mediaIDs {
		attachedMedia[i] = map[string]string{"media_fbid": id}
	}
	attachedJSON, _ := json.Marshal(attachedMedia)

	data := url.Values{
		"access_token":   {accessToken},
		"message":        {message},
		"attached_media": {string(attachedJSON)},
	}
	if targetingJSON != "" {
		data["targeting"] = []string{targetingJSON}
	}
	addScheduleParams(data, scheduleUnix)

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return handleResponse(resp)
}

// publishMultiPhoto orchestrates a multi-photo post: it uploads each image as an
// unpublished draft to get media IDs, then creates the final post with attached_media.
func publishMultiPhoto(images []ImageInput, message, accessToken, pageID, targetingJSON string, scheduleUnix int64) (string, error) {
	if scheduleUnix > 0 {
		theme.Info("Schedule", fmt.Sprintf("Scheduled for %s", time.Unix(scheduleUnix, 0).Format(time.RFC3339)))
	}
	theme.Info("Multi-photo", fmt.Sprintf("Uploading %d images as drafts...", len(images)))

	var mediaIDs []string
	for i, img := range images {
		theme.Info(fmt.Sprintf("  Draft %d/%d", i+1, len(images)), img.Filename)

		id, err := uploadPhotoDraft(pageID, img, accessToken)
		if err != nil {
			return "", fmt.Errorf("failed to upload draft image %q: %w", img.Filename, err)
		}
		mediaIDs = append(mediaIDs, id)

		if i < len(images)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	theme.Success(fmt.Sprintf("All %d drafts uploaded. Publishing multi-photo post...", len(mediaIDs)))
	return postMultiPhoto(pageID, message, mediaIDs, accessToken, targetingJSON, scheduleUnix)
}

func handleResponse(resp *http.Response) (string, error) {
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Status: %s, Body: %s", resp.Status, string(body))
	}

	var res map[string]any
	_ = json.Unmarshal(body, &res)

	if id, ok := res["id"].(string); ok {
		return id, nil
	}

	return string(body), nil
}
