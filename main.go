package main

import (
	theme "bubble/internal"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ImageList is a custom flag type to support multiple image flags
type ImageList []string

func (i *ImageList) String() string {
	return strings.Join(*i, ", ")
}

func (i *ImageList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type ImageInput struct {
	Path     string
	Type     string // "url" or "file"
	Size     int64
	Filename string
}

type UploadResult struct {
	Filename string
	Success  bool
	PostID   string
	Error    error
}

type ValidationResult struct {
	Valid     bool
	Images    []ImageInput
	Errors    []string
	Warnings  []string
	TotalSize int64
}

var client *http.Client

func main() {
	_ = godotenv.Load()

	// Load environment variables
	envPageID := os.Getenv("FB_PAGE_ID")
	envToken := os.Getenv("FB_ACCESS_TOKEN")
	envMessage := os.Getenv("FB_MESSAGE")
	envImages := os.Getenv("FB_IMAGES")

	var images ImageList
	messageFlag := flag.String("message", "", "The text message/caption to post (required if no images)")
	link := flag.String("link", "", "Optional URL to share as a link post")
	dryRun := flag.Bool("dry-run", false, "Validate inputs without uploading")
	rollback := flag.Bool("rollback", false, "Delete a specific post ID (provided as argument) or the latest if no ID given")
	timeout := flag.Duration("timeout", 60*time.Second, "HTTP request timeout")
	flag.Var(&images, "image", "Path to local image file or URL (can be specified multiple times)")
	flag.Parse()

	theme.PrintHeader()

	client = &http.Client{Timeout: *timeout}

	// Priority: CLI > ENV
	finalMessage := *messageFlag
	if finalMessage == "" {
		finalMessage = envMessage
	}

	allImagePaths := images
	if len(allImagePaths) == 0 && envImages != "" {
		paths := strings.SplitSeq(envImages, ",")
		for p := range paths {
			allImagePaths = append(allImagePaths, strings.TrimSpace(p))
		}
	}

	pageID := envPageID
	accessToken := envToken

	if pageID == "" || accessToken == "" {
		theme.Error("FB_PAGE_ID and FB_ACCESS_TOKEN environment variables must be set.")
		os.Exit(1)
	}

	if *rollback {
		targetID := ""
		if len(flag.Args()) > 0 {
			targetID = flag.Args()[0]
		}
		rollbackPost(pageID, targetID, accessToken)
		os.Exit(0)
	}

	if finalMessage == "" && len(allImagePaths) == 0 {
		theme.Error("Either -message or -image must be provided.")
		fmt.Printf("%sUsage: %s -message=\"Your message\" [-image=<path|url>] [-link=<url>] [-dry-run] [-rollback [post_id]]%s\n", theme.Gray, os.Args[0], theme.Reset)
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *link != "" && len(allImagePaths) > 0 {
		theme.Error("Cannot specify both -link and -image at the same time.")
		os.Exit(1)
	}

	// Detect and validate images
	parsedImages, valResult := validateInputs(allImagePaths, finalMessage)

	if *dryRun {
		printDryRunSummary(valResult)
		os.Exit(0)
	}

	if !valResult.Valid {
		theme.Error("Validation failed:")
		for _, err := range valResult.Errors {
			fmt.Printf("  %s- %s%s\n", theme.Gray, err, theme.Reset)
		}
		os.Exit(1)
	}

	if len(parsedImages) > 0 {
		results := uploadMultipleImages(parsedImages, finalMessage, accessToken, pageID)
		printResultsSummary(results)
	} else if *link != "" {
		postLink(pageID, finalMessage, *link, accessToken)
	} else {
		postText(pageID, finalMessage, accessToken)
	}
}

func validateInputs(imagePaths []string, message string) ([]ImageInput, *ValidationResult) {
	result := &ValidationResult{Valid: true}
	var parsed []ImageInput

	for _, path := range imagePaths {
		imgType := detectInputType(path)
		img := ImageInput{Path: path, Type: imgType}

		if imgType == "file" {
			info, err := os.Stat(path)
			if err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("File not found: %s", path))
				continue
			}
			img.Size = info.Size()
			img.Filename = filepath.Base(path)
			result.TotalSize += img.Size

			if img.Size > 8*1024*1024 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("File %s is large (%.2f MB), Facebook limit is 8MB for some types.", img.Filename, float64(img.Size)/1024/1024))
			}

			ext := strings.ToLower(filepath.Ext(path))
			validExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".tiff": true, ".webp": true, ".svg": true}
			if !validExts[ext] {
				result.Warnings = append(result.Warnings, fmt.Sprintf("File %s has potentially unsupported extension: %s", img.Filename, ext))
			}
		} else {
			img.Filename = extractFilenameFromURL(path)
		}
		parsed = append(parsed, img)
	}

	if message == "" && len(parsed) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Message or images must be provided")
	}

	result.Images = parsed
	return parsed, result
}

func detectInputType(input string) string {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return "url"
	}
	return "file"
}

func extractFilenameFromURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "image"
	}
	return filepath.Base(parsed.Path)
}

func printDryRunSummary(result *ValidationResult) {
	theme.PrintSection("Dry Run Validation")
	theme.Info("Valid", fmt.Sprintf("%v", result.Valid))
	theme.Info("Total Images", fmt.Sprintf("%d", len(result.Images)))
	theme.Info("Total Size", fmt.Sprintf("%.2f MB", float64(result.TotalSize)/1024/1024))

	fmt.Printf("\n%sImages to be processed:%s\n", theme.Gray, theme.Reset)
	for i, img := range result.Images {
		fmt.Printf("  %s%d. %s [%s]%s", theme.Gray, i+1, img.Filename, img.Type, theme.Reset)
		if img.Type == "file" {
			fmt.Printf(" %s(%.2f MB)%s", theme.Gray, float64(img.Size)/1024/1024, theme.Reset)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n%sErrors:%s\n", theme.Orange, theme.Reset)
		for _, e := range result.Errors {
			theme.Error(e)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("\n%sWarnings:%s\n", theme.Blue, theme.Reset)
		for _, w := range result.Warnings {
			theme.Warning(w)
		}
	}
}

func uploadMultipleImages(images []ImageInput, message, accessToken, pageID string) []UploadResult {
	var results []UploadResult

	theme.PrintSection(fmt.Sprintf("Uploading %d Images", len(images)))

	for i, img := range images {
		fmt.Printf("\n%s[%d/%d]%s %s%s%s\n", theme.Blue, i+1, len(images), theme.Reset, theme.Bold, img.Filename, theme.Reset)

		var postID string
		var err error

		if img.Type == "url" {
			postID, err = postImageURL(pageID, message, img.Path, accessToken)
		} else {
			postID, err = postImageFile(pageID, message, img.Path, accessToken)
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

func postImageURL(pageID, message, imageURL, accessToken string) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s/photos", pageID)
	data := url.Values{
		"access_token": {accessToken},
		"url":          {imageURL},
		"caption":      {message},
	}

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return handleResponse(resp)
}

func postImageFile(pageID, message, filePath, accessToken string) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s/photos", pageID)

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

func postLink(pageID, message, link, accessToken string) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s/feed", pageID)
	data := url.Values{
		"access_token": {accessToken},
		"message":      {message},
		"link":         {link},
	}

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		theme.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	id, err := handleResponse(resp)
	if err != nil {
		theme.Error(fmt.Sprintf("Failed: %v", err))
	} else {
		theme.Success(fmt.Sprintf("Published successfully! ID: %s", id))
	}
}

func postText(pageID, message, accessToken string) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s/feed", pageID)
	data := url.Values{
		"access_token": {accessToken},
		"message":      {message},
	}

	resp, err := client.PostForm(apiURL, data)
	if err != nil {
		theme.Error(fmt.Sprintf("%v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	id, err := handleResponse(resp)
	if err != nil {
		theme.Error(fmt.Sprintf("Failed: %v", err))
	} else {
		theme.Success(fmt.Sprintf("Published successfully! ID: %s", id))
	}
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

func printResultsSummary(results []UploadResult) {
	theme.PrintSection("Upload Summary")
	successCount := 0
	for _, res := range results {
		if res.Success {
			theme.Success(fmt.Sprintf("%s (ID: %s)", res.Filename, res.PostID))
			successCount++
		} else {
			theme.Error(fmt.Sprintf("%s (Error: %v)", res.Filename, res.Error))
		}
	}
	theme.Info("Total", fmt.Sprintf("%d", len(results)))
	theme.Info("Succeeded", fmt.Sprintf("%d", successCount))
	theme.Info("Failed", fmt.Sprintf("%d", len(results)-successCount))
}

func rollbackPost(pageID, postID, accessToken string) {
	targetID := postID

	if targetID == "" {
		theme.Info("Rollback", "No post ID provided. Fetching latest post...")

		apiURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s/feed?limit=1&access_token=%s", pageID, accessToken)
		resp, err := client.Get(apiURL)
		if err != nil {
			theme.Error(fmt.Sprintf("Error fetching feed: %v", err))
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			theme.Error(fmt.Sprintf("Failed to fetch feed. Status: %s, Body: %s", resp.Status, string(body)))
			os.Exit(1)
		}

		var feed struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &feed); err != nil {
			theme.Error(fmt.Sprintf("Error parsing feed JSON: %v", err))
			os.Exit(1)
		}

		if len(feed.Data) == 0 {
			theme.Warning("No posts found to rollback.")
			return
		}

		targetID = feed.Data[0].ID
		theme.Info("Target ID", targetID)
	} else {
		theme.Info("Target ID", targetID)
	}

	theme.Info("Status", "Deleting...")

	deleteURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s?access_token=%s", targetID, accessToken)
	req, _ := http.NewRequest("DELETE", deleteURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		theme.Error(fmt.Sprintf("Error sending delete request: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		theme.Success(fmt.Sprintf("Successfully deleted post ID: %s", targetID))
	} else {
		theme.Error(fmt.Sprintf("Failed to delete post. Status: %s, Body: %s", resp.Status, string(body)))
		os.Exit(1)
	}
}
