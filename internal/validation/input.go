// Package validation provides input validation, target resolution, and dry-run summaries.
package validation

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"facefeed/domain"
)

// ValidateInputs validates image paths and message, returning parsed images and validation result.
func ValidateInputs(imagePaths []string, message string) ([]domain.ImageInput, *domain.ValidationResult) {
	result := &domain.ValidationResult{Valid: true}
	var parsed []domain.ImageInput

	for _, path := range imagePaths {
		imgType := DetectInputType(path)
		img := domain.ImageInput{Path: path, Type: imgType}

		if imgType == domain.ImageTypeFile {
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
			img.Filename = ExtractFilenameFromURL(path)
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

// DetectInputType determines whether an input is a URL or a local file path.
func DetectInputType(input string) string {
	if input == "" {
		return domain.ImageTypeFile
	}
	parsed, err := url.Parse(input)
	if err != nil {
		return domain.ImageTypeFile
	}
	if parsed.Scheme != "" && parsed.Host != "" {
		return domain.ImageTypeURL
	}
	return domain.ImageTypeFile
}

// ExtractFilenameFromURL extracts the filename portion from a URL.
func ExtractFilenameFromURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "image"
	}
	return filepath.Base(parsed.Path)
}
