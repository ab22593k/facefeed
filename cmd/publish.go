package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	theme "facefeed/internal"
	"facefeed/internal/facebook"
	"facefeed/internal/validation"

	"facefeed/domain"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// publishCmd represents the publish command.
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish content to Facebook",
	Long: `Publish text, links, or images to one or more Facebook Pages or Groups.

You can specify targets via:
  - FB_PAGE_ID environment variable (single page)
  - --groups flag (one or more group IDs)
  - --config flag (JSON file with per-target messages)

Schedule a post with --schedule:
  --schedule "2025-01-01T15:00:00Z"   ISO 8601 datetime
  --schedule 1735689600                Unix timestamp
  --schedule "2h"                      Relative time (30m, 2h, 7d)

Examples:
  facefeed publish --message "Hello, world!"
  facefeed publish --message "Check this out" --link "https://example.com"
  facefeed publish --image photo.jpg --message "My photo"
  facefeed publish --image photo1.jpg --image photo2.jpg --multi-photo --message "Gallery"
  facefeed publish --image url1.jpg --image url2.jpg --groups group1,group2
  facefeed publish --config targets.json --message "Default message"
  facefeed publish --message "US only" --targeting '{"geo_locations":{"countries":["US"]}}'
  facefeed publish --message "Tomorrow" --schedule "24h" --targeting @targeting.json`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load()

		envPageID := os.Getenv("FB_PAGE_ID")
		envToken := os.Getenv("FB_ACCESS_TOKEN")
		envMessage := os.Getenv("FB_MESSAGE")
		envImages := os.Getenv("FB_IMAGES")

		messageFlag, _ := cmd.Flags().GetString("message")
		groupsFlag, _ := cmd.Flags().GetString("groups")
		configPath, _ := cmd.Flags().GetString("config")
		link, _ := cmd.Flags().GetString("link")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		images, _ := cmd.Flags().GetStringSlice("image")
		targetingRaw, _ := cmd.Flags().GetString("targeting")
		multiPhoto, _ := cmd.Flags().GetBool("multi-photo")
		scheduleRaw, _ := cmd.Flags().GetString("schedule")

		// Parse targeting
		targetingJSON, err := validation.ParseTargeting(targetingRaw)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to parse targeting: %v", err))
			os.Exit(1)
		}

		// Parse schedule time
		scheduleUnix, err := validation.ParseSchedule(scheduleRaw)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to parse schedule time: %v", err))
			os.Exit(1)
		}

		// Resolve Targets
		targets, err := validation.ResolveTargets(envPageID, groupsFlag, configPath)
		if err != nil {
			theme.Error(fmt.Sprintf("Failed to resolve targets: %v", err))
			os.Exit(1)
		}

		if envToken == "" {
			theme.Error("FB_ACCESS_TOKEN environment variable must be set.")
			os.Exit(1)
		}

		if len(targets) == 0 {
			theme.Error("No targets specified. Set FB_PAGE_ID, use --groups, or --config.")
			os.Exit(1)
		}

		// Priority: CLI > ENV
		finalMessage := messageFlag
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

		if finalMessage == "" && len(allImagePaths) == 0 && configPath == "" {
			theme.Error("Either --message, --image, or --config must be provided.")
			fmt.Printf("%sUsage: facefeed publish --message=\"Your message\" [--image=<path|url>] [--link=<url>] [--groups=id1,id2] [--config=targets.json] [--targeting=<json|@file>] [--multi-photo] [--schedule=<time>] [--dry-run]%s\n", theme.Gray, theme.Reset)
			os.Exit(1)
		}

		if link != "" && len(allImagePaths) > 0 {
			theme.Error("Cannot specify both --link and --image at the same time.")
			os.Exit(1)
		}

		if multiPhoto && len(allImagePaths) < 2 {
			theme.Error("--multi-photo requires at least 2 images, provided with --image.")
			os.Exit(1)
		}

		if multiPhoto && configPath != "" {
			theme.Info("Multi-photo", "Using per-target messages from config will apply the same media across all targets")
		}

		// Validate inputs
		parsedImages, valResult := validation.ValidateInputs(allImagePaths, finalMessage)

		if dryRun {
			validation.PrintDryRunSummary(valResult, targets)
			if targetingJSON != "" {
				theme.Info("Targeting", targetingJSON)
			}
			if multiPhoto {
				theme.Info("Mode", "Multi-photo gallery (all images in one post)")
			}
			if scheduleUnix > 0 {
				theme.Info("Schedule", time.Unix(scheduleUnix, 0).Format(time.RFC3339))
			}
			os.Exit(0)
		}

		if !valResult.Valid {
			theme.Error("Validation failed:")
			for _, err := range valResult.Errors {
				fmt.Printf("  %s- %s%s\n", theme.Gray, err, theme.Reset)
			}
			os.Exit(1)
		}

		// Batch Execution
		var allTargetResults []domain.TargetResult
		for _, target := range targets {
			theme.PrintSection(fmt.Sprintf("Target: %s", target.ID))

			targetMessage := target.Message
			if targetMessage == "" {
				targetMessage = finalMessage
			}

			targetRes := domain.TargetResult{TargetID: target.ID}

			if multiPhoto && len(parsedImages) > 0 {
				postID, err := facebook.PublishMultiPhoto(parsedImages, targetMessage, envToken, target.ID, targetingJSON, scheduleUnix)
				targetRes.PostID = postID
				targetRes.Error = err
				if err != nil {
					theme.Error(fmt.Sprintf("Multi-photo post failed: %v", err))
				} else {
					theme.Success(fmt.Sprintf("Multi-photo post published! ID: %s", postID))
				}
			} else if len(parsedImages) > 0 {
				results := facebook.UploadMultipleImages(parsedImages, targetMessage, envToken, target.ID, targetingJSON, scheduleUnix)
				targetRes.Results = results
			} else if link != "" {
				postID, err := facebook.PostLink(target.ID, targetMessage, link, envToken, targetingJSON, scheduleUnix)
				targetRes.PostID = postID
				targetRes.Error = err
				if err != nil {
					theme.Error(fmt.Sprintf("Failed: %v", err))
				} else {
					theme.Success(fmt.Sprintf("Published successfully! ID: %s", postID))
				}
			} else {
				postID, err := facebook.PostText(target.ID, targetMessage, envToken, targetingJSON, scheduleUnix)
				targetRes.PostID = postID
				targetRes.Error = err
				if err != nil {
					theme.Error(fmt.Sprintf("Failed: %v", err))
				} else {
					theme.Success(fmt.Sprintf("Published successfully! ID: %s", postID))
				}
			}
			allTargetResults = append(allTargetResults, targetRes)
		}

		validation.PrintBatchResultsSummary(allTargetResults)
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().String("message", "", "The text message/caption to post")
	publishCmd.Flags().StringSlice("image", nil, "Path to local image file or URL (can be specified multiple times)")
	publishCmd.Flags().String("link", "", "Optional URL to share as a link post")
	publishCmd.Flags().String("groups", "", "Comma-separated list of Group IDs to publish to")
	publishCmd.Flags().String("config", "", "Path to JSON config file for batch publishing with per-group messages")
	publishCmd.Flags().Bool("dry-run", false, "Validate inputs without uploading")
	publishCmd.Flags().String("targeting", "", `Audience targeting JSON (inline or @filepath). Example: '{"geo_locations":{"countries":["US"]}}'`)
	publishCmd.Flags().Bool("multi-photo", false, "Combine multiple --image values into a single multi-photo post (requires 2+ images)")
	publishCmd.Flags().String("schedule", "", `Schedule post for a future time. Formats: Unix timestamp, ISO 8601 (e.g. "2025-01-01T15:00:00Z"), or relative ("30m", "2h", "7d")`)
}
