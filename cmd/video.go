package cmd

import (
	"fmt"
	"os"

	theme "facefeed/internal"
	"facefeed/internal/presentation"
	"facefeed/internal/validation"

	"github.com/spf13/cobra"
)

// videoCmd represents the video command.
var videoCmd = &cobra.Command{
	Use:   "video [file_path]",
	Short: "Publish a video to Facebook",
	Long: `Upload and publish a video to a Facebook Page feed.

The video file is uploaded as a multipart upload to the Page's videos endpoint.
Supports scheduling.

Examples:
  facefeed video my_video.mp4 --title "My Video" --description "A great video"
  facefeed video my_video.mp4 --title "Scheduled" --schedule "2h"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envPageID := os.Getenv("FB_PAGE_ID")
		groupsFlag, _ := cmd.Flags().GetString("groups")
		configPath, _ := cmd.Flags().GetString("config")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		scheduleRaw, _ := cmd.Flags().GetString("schedule")

		scheduleUnix, err := validation.ParseSchedule(scheduleRaw)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to parse schedule: %v", err))
			os.Exit(1)
		}

		targets, err := validation.ResolveTargets(envPageID, groupsFlag, configPath)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to resolve targets: %v", err))
			os.Exit(1)
		}
		if len(targets) == 0 {
			theme.Error("No targets specified. Set FB_PAGE_ID, use --groups, or --config.")
			os.Exit(1)
		}

		filePath := args[0]
		theme.PrintSection("Video Upload")

		for _, target := range targets {
			theme.Info("Target", target.ID)
			postID, err := FBClient.PostVideoUpload(target.ID, title, description, filePath, scheduleUnix)
			presentation.DisplayPublishResult("Video", postID, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)

	videoCmd.Flags().String("title", "", "Title of the video")
	videoCmd.Flags().String("description", "", "Description of the video")
	videoCmd.Flags().String("schedule", "", `Schedule time: Unix timestamp, ISO 8601, or relative ("2h", "7d")`)
	videoCmd.Flags().String("groups", "", "Comma-separated list of Group IDs")
	videoCmd.Flags().String("config", "", "Path to JSON config file for per-target publishing")
}
