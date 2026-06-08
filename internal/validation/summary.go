package validation

import (
	"fmt"

	theme "facefeed/internal"

	"facefeed/domain"
)

// PrintDryRunSummary prints a dry-run validation summary to the user.
func PrintDryRunSummary(result *domain.ValidationResult, targets []domain.PublishTarget) {
	theme.PrintSection("Dry Run Validation")
	theme.Info("Valid", fmt.Sprintf("%v", result.Valid))
	theme.Info("Targets", fmt.Sprintf("%d", len(targets)))
	for i, t := range targets {
		msg := "Default"
		if t.Message != "" {
			msg = "Custom"
		}
		fmt.Printf("  %s%d. %s [%s message]%s\n", theme.Gray, i+1, t.ID, msg, theme.Reset)
	}
	theme.Info("Total Images", fmt.Sprintf("%d", len(result.Images)))
	theme.Info("Total Size", fmt.Sprintf("%.2f MB", float64(result.TotalSize)/1024/1024))

	fmt.Printf("\n%sImages to be processed:%s\n", theme.Gray, theme.Reset)
	for i, img := range result.Images {
		fmt.Printf("  %s%d. %s [%s]%s", theme.Gray, i+1, img.Filename, img.Type, theme.Reset)
		if img.Type == domain.ImageTypeFile {
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

// PrintBatchResultsSummary prints a batch publish results summary to the user.
func PrintBatchResultsSummary(results []domain.TargetResult) {
	theme.PrintSection("Batch Results Summary")
	for _, res := range results {
		if len(res.Results) > 0 {
			successCount := 0
			for _, r := range res.Results {
				if r.Success {
					successCount++
				}
			}
			if successCount == len(res.Results) {
				theme.Success(fmt.Sprintf("Target %s: All %d images uploaded.", res.TargetID, len(res.Results)))
			} else {
				theme.Warning(fmt.Sprintf("Target %s: %d/%d images uploaded successfully.", res.TargetID, successCount, len(res.Results)))
			}
		} else {
			if res.Error == nil {
				theme.Success(fmt.Sprintf("Target %s: Published successfully (ID: %s)", res.TargetID, res.PostID))
			} else {
				theme.Error(fmt.Sprintf("Target %s: Failed (%v)", res.TargetID, res.Error))
			}
		}
	}
}
