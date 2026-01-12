package main

import (
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
	"github.com/schollz/progressbar/v3"
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

	client = &http.Client{Timeout: *timeout}

	// Priority: CLI > ENV
	finalMessage := *messageFlag
	if finalMessage == "" {
		finalMessage = envMessage
	}

	allImagePaths := images
	if len(allImagePaths) == 0 && envImages != "" {
		paths := strings.Split(envImages, ",")
		for _, p := range paths {
			allImagePaths = append(allImagePaths, strings.TrimSpace(p))
		}
	}

	pageID := envPageID
	accessToken := envToken

	if pageID == "" || accessToken == "" {
		fmt.Println("Error: FB_PAGE_ID and FB_ACCESS_TOKEN environment variables must be set.")
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
		fmt.Println("Error: Either -message or -image must be provided.")
		fmt.Printf("Usage: %s -message=\"Your message\" [-image=<path|url>] [-link=<url>] [-dry-run] [-rollback [post_id]]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *link != "" && len(allImagePaths) > 0 {
		fmt.Println("Error: Cannot specify both -link and -image at the same time.")
		os.Exit(1)
	}

	// Detect and validate images
	parsedImages, valResult := validateInputs(allImagePaths, finalMessage)

	if *dryRun {
		printDryRunSummary(valResult)
		os.Exit(0)
	}

	if !valResult.Valid {
		fmt.Println("Validation failed:")
		for _, err := range valResult.Errors {
			fmt.Printf("  - %s\n", err)
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
	fmt.Println("=== DRY RUN VALIDATION ===")
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Total Images: %d\n", len(result.Images))
	fmt.Printf("Total File Size: %.2f MB\n", float64(result.TotalSize)/1024/1024)

	fmt.Println("\nImages to be processed:")
	for i, img := range result.Images {
		fmt.Printf("  %d. %s [%s]", i+1, img.Filename, img.Type)
		if img.Type == "file" {
			fmt.Printf(" (%.2f MB)", float64(img.Size)/1024/1024)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}

func uploadMultipleImages(images []ImageInput, message, accessToken, pageID string) []UploadResult {
	var results []UploadResult

	fmt.Printf("Starting upload of %d images...\n", len(images))

	for i, img := range images {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(images), img.Filename)

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
			fmt.Println("Waiting 1s before next upload to avoid rate limits...")
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
		// Check if ImageMagick's convert command exists
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

	// Create a pipe to stream the upload and track progress
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

		// Progress bar
		bar := progressbar.DefaultBytes(
			info.Size(),
			"uploading",
		)

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
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	id, err := handleResponse(resp)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	} else {
		fmt.Printf("Published successfully! ID: %s\n", id)
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
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	id, err := handleResponse(resp)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	} else {
		fmt.Printf("Published successfully! ID: %s\n", id)
	}
}

func handleResponse(resp *http.Response) (string, error) {
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Status: %s, Body: %s", resp.Status, string(body))
	}

	var res map[string]interface{}
	_ = json.Unmarshal(body, &res)

	if id, ok := res["id"].(string); ok {
		return id, nil
	}

	return string(body), nil
}

func printResultsSummary(results []UploadResult) {
	fmt.Println("\n=== UPLOAD SUMMARY ===")
	successCount := 0
	for _, res := range results {
		if res.Success {
			fmt.Printf("✓ %s (ID: %s)\n", res.Filename, res.PostID)
			successCount++
		} else {
			fmt.Printf("✗ %s (Error: %v)\n", res.Filename, res.Error)
		}
	}
	fmt.Printf("\nTotal: %d, Succeeded: %d, Failed: %d\n", len(results), successCount, len(results)-successCount)
}

func rollbackPost(pageID, postID, accessToken string) {
	targetID := postID

	if targetID == "" {
		fmt.Println("No post ID provided. Attempting to rollback latest post...")

		// 1. Fetch latest post ID
		apiURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s/feed?limit=1&access_token=%s", pageID, accessToken)
		resp, err := client.Get(apiURL)
		if err != nil {
			fmt.Printf("Error fetching feed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Failed to fetch feed. Status: %s, Body: %s\n", resp.Status, string(body))
			os.Exit(1)
		}

		var feed struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &feed); err != nil {
			fmt.Printf("Error parsing feed JSON: %v\n", err)
			os.Exit(1)
		}

		if len(feed.Data) == 0 {
			fmt.Println("No posts found to rollback.")
			return
		}

		targetID = feed.Data[0].ID
		fmt.Printf("Found latest post ID: %s\n", targetID)
	} else {
		fmt.Printf("Attempting to rollback specific post ID: %s\n", targetID)
	}

	fmt.Println("Deleting...")

	// 2. Delete the post
	deleteURL := fmt.Sprintf("https://graph.facebook.com/v24.0/%s?access_token=%s", targetID, accessToken)
	req, _ := http.NewRequest("DELETE", deleteURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending delete request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully deleted post ID: %s\n", targetID)
	} else {
		fmt.Printf("Failed to delete post. Status: %s, Body: %s\n", resp.Status, string(body))
		os.Exit(1)
	}
}
