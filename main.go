package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	pageID := os.Getenv("FB_PAGE_ID")
	accessToken := os.Getenv("FB_ACCESS_TOKEN")

	message := flag.String("message", "", "The text message/caption to post (required)")
	link := flag.String("link", "", "Optional URL to share as a link post")
	photo := flag.String("photo", "", "Optional public URL to an image to post as a photo")
	flag.Parse()

	if pageID == "" || accessToken == "" || *message == "" {
		fmt.Println("Error: FB_PAGE_ID and FB_ACCESS_TOKEN environment variables must be set.")
		fmt.Println("Usage: go run main.go -message=\"Your message\" [-link=<url> | -photo=<image_url>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *link != "" && *photo != "" {
		fmt.Println("Error: Cannot specify both -link and -photo at the same time.")
		os.Exit(1)
	}

	apiVersion := "v24.0"
	endpoint := "feed" // default for text or link posts

	data := url.Values{
		"access_token": {accessToken},
	}

	if *photo != "" {
		endpoint = "photos"
		data.Set("url", *photo)
		data.Set("caption", *message)
	} else {
		data.Set("message", *message)
		if *link != "" {
			data.Set("link", *link)
		}
	}

	apiURL := fmt.Sprintf("https://graph.facebook.com/%s/%s/%s", apiVersion, pageID, endpoint)

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Post published successfully!")
		fmt.Printf("Response: %s\n", string(body)) // Contains the post ID
	} else {
		fmt.Printf("Failed to publish post. Status: %s\nResponse: %s\n", resp.Status, string(body))
	}
}
