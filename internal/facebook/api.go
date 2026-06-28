// Package facebook implements the Facebook Graph API client for publishing content.
package facebook

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	theme "facefeed/internal"

	"facefeed/domain"
)

// UploadMultipleImages uploads multiple images to a Facebook page, returning results for each.
func (c *fbClient) UploadMultipleImages(images []domain.ImageInput, message, pageID, targetingJSON string, scheduleUnix int64) []domain.UploadResult {
	var results []domain.UploadResult

	if scheduleUnix > 0 {
		theme.Info("Schedule", fmt.Sprintf("Scheduled for %s", time.Unix(scheduleUnix, 0).Format(time.RFC3339)))
	}
	theme.PrintSection(fmt.Sprintf("Uploading %d Images", len(images)))

	for i, img := range images {
		fmt.Printf("\n%s[%d/%d]%s %s%s%s\n", theme.Blue, i+1, len(images), theme.Reset, theme.Bold, img.Filename, theme.Reset)

		var postID string
		var err error

		if img.Type == domain.ImageTypeURL {
			postID, err = c.PostImageURL(pageID, message, img.Path, targetingJSON, scheduleUnix)
		} else {
			postID, err = c.PostImageFile(pageID, message, img.Path, targetingJSON, scheduleUnix)
		}

		results = append(results, domain.UploadResult{
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

// PostImageURL posts an image from a URL to a Facebook page.
func (c *fbClient) PostImageURL(pageID, message, imageURL, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(pageID + "/photos")
	data := map[string]string{
		paramAccessToken: c.accessToken, paramURL: imageURL,
		"caption": message,
	}
	if targetingJSON != "" {
		data["targeting"] = targetingJSON
	}
	addScheduleParams(data, scheduleUnix)

	return c.postForm(apiURL, data)
}

// convertSVGToPNG converts an SVG file to a temporary PNG file and returns the PNG path.
// The caller is responsible for removing the returned file when done.
func convertSVGToPNG(filePath string) (pngPath string, err error) {
	if _, err := exec.LookPath("convert"); err != nil {
		return "", fmt.Errorf("SVG files require ImageMagick (convert command not found).\nInstall: apt-get install imagemagick (Linux) or brew install imagemagick (macOS)")
	}
	tempPNG := filePath + ".temp.png"
	cmd := exec.Command("convert", filePath, tempPNG)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tempPNG)
		return "", fmt.Errorf("failed to convert SVG to PNG: %w", err)
	}
	return tempPNG, nil
}

// PostImageFile uploads a local image file to a Facebook page, converting SVGs to PNGs.
func (c *fbClient) PostImageFile(pageID, message, filePath, targetingJSON string, scheduleUnix int64) (string, error) {
	apiURL := GraphAPIURL(pageID + "/photos")

	uploadPath := filePath
	if strings.ToLower(filepath.Ext(filePath)) == ".svg" {
		pngPath, err := convertSVGToPNG(filePath)
		if err != nil {
			return "", err
		}
		uploadPath = pngPath
		defer func() { _ = os.Remove(pngPath) }()
	}

	file, err := os.Open(uploadPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	info, _ := file.Stat()

	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)

	errChan := make(chan error, 1)

	go func() {
		defer func() { _ = bodyWriter.Close() }()
		defer func() { _ = multiWriter.Close() }()
		defer close(errChan)

		_ = multiWriter.WriteField(paramAccessToken, c.accessToken)
		_ = multiWriter.WriteField("caption", message)
		if targetingJSON != "" {
			_ = multiWriter.WriteField("targeting", targetingJSON)
		}
		if scheduleUnix > 0 {
			_ = multiWriter.WriteField("published", "false")
			_ = multiWriter.WriteField("scheduled_publish_time", fmt.Sprintf("%d", scheduleUnix))
		}

		part, gErr := multiWriter.CreateFormFile("source", filepath.Base(uploadPath))
		if gErr != nil {
			errChan <- gErr
			return
		}

		bar := theme.NewProgressBar(info.Size(), "uploading")

		_, gErr = io.Copy(io.MultiWriter(part, bar), file)
		if gErr != nil {
			errChan <- gErr
			return
		}
	}()

	req, err := http.NewRequest("POST", apiURL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	gErr := <-errChan
	if gErr != nil {
		return "", gErr
	}

	return HandleResponse(resp)
}

// UploadPhotoDraft uploads a single image as an unpublished draft and returns its media ID.
// It handles both URL and file-based images, including SVG-to-PNG conversion.
func (c *fbClient) UploadPhotoDraft(pageID string, img domain.ImageInput) (string, error) {
	if img.Type == domain.ImageTypeURL {
		apiURL := GraphAPIURL(pageID + "/photos")
		data := map[string]string{
			paramAccessToken: c.accessToken, paramURL: img.Path,
			"published": "false",
		}

		return c.postForm(apiURL, data)
	}

	// File-based upload (multipart)
	apiURL := GraphAPIURL(pageID + "/photos")

	uploadPath := img.Path
	if strings.ToLower(filepath.Ext(img.Path)) == ".svg" {
		pngPath, err := convertSVGToPNG(img.Path)
		if err != nil {
			return "", err
		}
		uploadPath = pngPath
		defer func() { _ = os.Remove(pngPath) }()
	}

	file, err := os.Open(uploadPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)

	errChan := make(chan error, 1)

	go func() {
		defer func() { _ = bodyWriter.Close() }()
		defer func() { _ = multiWriter.Close() }()
		defer close(errChan)

		_ = multiWriter.WriteField(paramAccessToken, c.accessToken)
		_ = multiWriter.WriteField("published", "false")

		part, gErr := multiWriter.CreateFormFile("source", filepath.Base(uploadPath))
		if gErr != nil {
			errChan <- gErr
			return
		}

		_, gErr = io.Copy(part, file)
		if gErr != nil {
			errChan <- gErr
			return
		}
	}()

	req, err := http.NewRequest("POST", apiURL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	gErr := <-errChan
	if gErr != nil {
		return "", gErr
	}

	mediaID, draftErr := HandleResponse(resp)
	if draftErr != nil {
		return "", fmt.Errorf("failed to upload draft %s: %w", img.Filename, draftErr)
	}
	return mediaID, nil
}

// PublishMultiPhoto orchestrates a multi-photo post: it uploads each image as an
// unpublished draft to get media IDs, then creates the final post with attached_media.
func (c *fbClient) PublishMultiPhoto(images []domain.ImageInput, message, pageID, targetingJSON string, scheduleUnix int64) (string, error) {
	if scheduleUnix > 0 {
		theme.Info("Schedule", fmt.Sprintf("Scheduled for %s", time.Unix(scheduleUnix, 0).Format(time.RFC3339)))
	}
	theme.Info("Multi-photo", fmt.Sprintf("Uploading %d images as drafts...", len(images)))

	var mediaIDs []string
	for i, img := range images {
		theme.Info(fmt.Sprintf("  Draft %d/%d", i+1, len(images)), img.Filename)

		id, err := c.UploadPhotoDraft(pageID, img)
		if err != nil {
			return "", fmt.Errorf("failed to upload draft image %q: %w", img.Filename, err)
		}
		mediaIDs = append(mediaIDs, id)

		if i < len(images)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	theme.Success(fmt.Sprintf("All %d drafts uploaded. Publishing multi-photo post...", len(mediaIDs)))
	return c.PostMultiPhoto(pageID, message, mediaIDs, targetingJSON, scheduleUnix)
}
