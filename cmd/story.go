package cmd

import (
	"os"
	"path/filepath"
	"strings"

	theme "facefeed/internal"
	"facefeed/internal/presentation"
	"facefeed/internal/validation"

	"github.com/spf13/cobra"
)

// storyCmd represents the story command.
var storyCmd = &cobra.Command{
	Use:   "story [file_path]",
	Short: "Publish a Story to Facebook",
	Long: `Upload and publish a photo or video as a Facebook Story.

Stories are temporary posts that disappear after 24 hours.
Supports image files (jpg, png) and video files (mp4).

Examples:
  facefeed story my_photo.jpg
  facefeed story my_video.mp4`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envPageID := os.Getenv("FB_PAGE_ID")
		envToken := os.Getenv("FB_ACCESS_TOKEN")

		if envPageID == "" {
			theme.Error("FB_PAGE_ID environment variable must be set.")
			os.Exit(1)
		}
		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		filePath := args[0]

		// Check file exists
		imgType := validation.DetectInputType(filePath)
		if imgType != "file" {
			theme.Error("Story command requires a local file path.")
			os.Exit(1)
		}

		// Detect if video based on extension
		ext := strings.ToLower(filepath.Ext(filePath))
		isVideo := ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".webm"

		contentType := "Photo Story"
		if isVideo {
			contentType = "Video Story"
		}
		theme.PrintSection(contentType)

		theme.Info("Target", envPageID)
		postID, err := FBClient.PublishStory(envPageID, filePath, isVideo)
		presentation.DisplayPublishResult(contentType, postID, err)
	},
}

func init() {
	rootCmd.AddCommand(storyCmd)
}
