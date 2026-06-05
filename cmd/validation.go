package cmd

import (
	theme "bubble/internal"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func resolveTargets(pageID, groups, configPath string) ([]PublishTarget, error) {
	var targets []PublishTarget

	if configPath != "" {
		file, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("could not read config file: %w", err)
		}
		if err := json.Unmarshal(file, &targets); err != nil {
			return nil, fmt.Errorf("could not parse config JSON: %w", err)
		}
		return targets, nil
	}

	if pageID != "" {
		targets = append(targets, PublishTarget{ID: pageID})
	}

	if groups != "" {
		groupIDs := strings.Split(groups, ",")
		for _, id := range groupIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				targets = append(targets, PublishTarget{ID: id})
			}
		}
	}

	return targets, nil
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

// parseSchedule converts a schedule time string to a Unix timestamp.
// It supports:
//   - Unix timestamps (numeric, seconds)
//   - ISO 8601 datetime strings (e.g. "2025-01-01T15:00:00Z")
//   - Relative durations ("30m", "2h", "1d", "7d")
func parseSchedule(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}

	// Try Unix timestamp first (all digits).
	if ts, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if ts < time.Now().Unix() {
			return 0, fmt.Errorf("schedule time %d is in the past", ts)
		}
		return ts, nil
	}

	// Try ISO 8601 formats.
	isoFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range isoFormats {
		if t, err := time.Parse(f, raw); err == nil {
			if t.Before(time.Now()) {
				return 0, fmt.Errorf("schedule time %s is in the past", raw)
			}
			return t.Unix(), nil
		}
	}

	// Try relative durations: "30m", "2h", "1d", "7d"
	if len(raw) >= 2 {
		durationStr := raw[:len(raw)-1]
		unit := raw[len(raw)-1]
		if val, err := strconv.ParseInt(durationStr, 10, 64); err == nil && val > 0 {
			var d time.Duration
			switch unit {
			case 'm':
				d = time.Duration(val) * time.Minute
			case 'h':
				d = time.Duration(val) * time.Hour
			case 'd':
				d = time.Duration(val) * 24 * time.Hour
			default:
				return 0, fmt.Errorf("unsupported duration unit %q (use m, h, or d)", unit)
			}
			return time.Now().Add(d).Unix(), nil
		}
	}

	return 0, fmt.Errorf("could not parse schedule time %q: use Unix timestamp, ISO 8601, or relative duration (30m, 2h, 1d)", raw)
}

// parseTargeting accepts either a JSON string or a file path prefixed with "@filename"
// and returns the validated targeting JSON string (or empty if not provided).
func parseTargeting(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	var rawJSON string
	if strings.HasPrefix(raw, "@") {
		path := raw[1:]
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("could not read targeting file %s: %w", path, err)
		}
		rawJSON = string(data)
	} else {
		rawJSON = raw
	}

	// Validate that it's proper JSON.
	var js any
	if err := json.Unmarshal([]byte(rawJSON), &js); err != nil {
		return "", fmt.Errorf("invalid targeting JSON: %w", err)
	}

	return rawJSON, nil
}

func printDryRunSummary(result *ValidationResult, targets []PublishTarget) {
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

func printBatchResultsSummary(results []TargetResult) {
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
