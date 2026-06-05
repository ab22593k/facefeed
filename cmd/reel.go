package cmd

import (
	"os"

	theme "facefeed/internal"
	"facefeed/internal/presentation"

	"github.com/spf13/cobra"
)

// reelCmd represents the reel command.
var reelCmd = &cobra.Command{
	Use:   "reel [file_path]",
	Short: "Publish a Reel to Facebook",
	Long: `Upload and publish a short-form video as a Facebook Reel.

Reels are short, entertaining videos on Facebook. The ideal aspect ratio
is 9:16 (vertical), and the recommended duration is 3-90 seconds.

Examples:
  facefeed reel my_reel.mp4 --description "Fun reel!"
  facefeed reel reel.mp4 --description "With hashtags #fun #reel"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envPageID := os.Getenv("FB_PAGE_ID")
		description, _ := cmd.Flags().GetString("description")

		if envPageID == "" {
			theme.Error("FB_PAGE_ID environment variable must be set.")
			os.Exit(1)
		}

		filePath := args[0]
		theme.PrintSection("Reel Publish")

		theme.Info("Target", envPageID)
		postID, err := FBClient.PublishReel(envPageID, description, filePath)
		presentation.DisplayPublishResult("Reel", postID, err)
	},
}

func init() {
	rootCmd.AddCommand(reelCmd)

	reelCmd.Flags().String("description", "", "Description/caption for the Reel")
}
