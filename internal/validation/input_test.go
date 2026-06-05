package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"facefeed/domain"
)

func TestValidateInputs(t *testing.T) {
	t.Parallel()

	// Set up temp files for file-based tests.
	tmpDir := t.TempDir()

	validJPEG := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(validJPEG, []byte("fake jpeg data"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	validPNG := filepath.Join(tmpDir, "image.png")
	if err := os.WriteFile(validPNG, []byte("fake png data"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	unsupportedExt := filepath.Join(tmpDir, "document.heic")
	if err := os.WriteFile(unsupportedExt, []byte("fake heic data"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Create a file larger than 8 MB for the large-file test.
	largeFilePath := filepath.Join(tmpDir, "large.png")
	largeContent := make([]byte, 9*1024*1024)
	if err := os.WriteFile(largeFilePath, largeContent, 0644); err != nil {
		t.Fatalf("failed to write large temp file: %v", err)
	}

	nonexistentPath := filepath.Join(tmpDir, "nonexistent.jpg")

	tests := []struct {
		name       string
		imagePaths []string
		message    string
		wantValid  bool
		wantImages int
		check      func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult)
	}{
		{
			name:       "empty message and no images is invalid",
			wantValid:  false,
			wantImages: 0,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				if len(result.Errors) == 0 {
					t.Error("expected error about missing message/images, got none")
				}
			},
		},
		{
			name:      "message only with no images is valid",
			message:   "Hello, world!",
			wantValid: true,
		},
		{
			name:       "URL image is valid",
			imagePaths: []string{"https://example.com/image.jpg"},
			message:    "Check this out",
			wantValid:  true,
			wantImages: 1,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				if images[0].Type != "url" {
					t.Errorf("image type = %q, want %q", images[0].Type, "url")
				}
				if images[0].Filename != "image.jpg" {
					t.Errorf("filename = %q, want %q", images[0].Filename, "image.jpg")
				}
			},
		},
		{
			name:       "multiple URLs",
			imagePaths: []string{"https://example.com/a.jpg", "https://example.com/b.png"},
			message:    "Gallery post",
			wantValid:  true,
			wantImages: 2,
		},
		{
			name:       "valid local file",
			imagePaths: []string{validJPEG},
			message:    "My photo",
			wantValid:  true,
			wantImages: 1,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				if images[0].Type != "file" {
					t.Errorf("image type = %q, want %q", images[0].Type, "file")
				}
				if images[0].Size <= 0 {
					t.Error("file size should be > 0")
				}
				if images[0].Filename != "photo.jpg" {
					t.Errorf("filename = %q, want %q", images[0].Filename, "photo.jpg")
				}
			},
		},
		{
			name:       "nonexistent file reports error",
			imagePaths: []string{nonexistentPath},
			message:    "Missing file",
			wantValid:  false,
			wantImages: 0,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err, "File not found") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected 'File not found' error, got errors: %v", result.Errors)
				}
			},
		},
		{
			name:       "unsupported extension triggers warning",
			imagePaths: []string{unsupportedExt},
			message:    "Unsupported format",
			wantValid:  true,
			wantImages: 1,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				found := false
				for _, w := range result.Warnings {
					if strings.Contains(w, "unsupported extension") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning about unsupported extension, got warnings: %v", result.Warnings)
				}
			},
		},
		{
			name:       "large file triggers size warning",
			imagePaths: []string{largeFilePath},
			message:    "Large file test",
			wantValid:  true,
			wantImages: 1,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				if len(result.Warnings) == 0 {
					t.Error("expected warning for large file, got none")
				}
				if result.TotalSize <= 8*1024*1024 {
					t.Errorf("total size = %d, expected > 8MB", result.TotalSize)
				}
			},
		},
		{
			name:       "mixed URL and local file",
			imagePaths: []string{"https://example.com/photo.jpg", validJPEG},
			message:    "Mixed sources",
			wantValid:  true,
			wantImages: 2,
			check: func(t *testing.T, images []domain.ImageInput, result *domain.ValidationResult) {
				if images[0].Type != "url" {
					t.Errorf("images[0].Type = %q, want %q", images[0].Type, "url")
				}
				if images[1].Type != "file" {
					t.Errorf("images[1].Type = %q, want %q", images[1].Type, "file")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, result := ValidateInputs(tt.imagePaths, tt.message)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v (errors: %v, warnings: %v)", result.Valid, tt.wantValid, result.Errors, result.Warnings)
			}

			if len(images) != tt.wantImages {
				t.Errorf("got %d images, want %d", len(images), tt.wantImages)
			}

			if tt.check != nil {
				tt.check(t, images, result)
			}
		})
	}
}

func TestDetectInputType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "https://example.com/image.jpg", want: "url"},
		{input: "http://example.com/image.jpg", want: "url"},
		{input: "/path/to/local/file.jpg", want: "file"},
		{input: "relative/path.png", want: "file"},
		{input: "photo.jpg", want: "file"},
		{input: "", want: "file"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DetectInputType(tt.input)
			if got != tt.want {
				t.Errorf("DetectInputType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractFilenameFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want string
	}{
		{url: "https://example.com/image.jpg", want: "image.jpg"},
		{url: "https://example.com/path/to/photo.png", want: "photo.png"},
		{url: "https://example.com/image.jpg?query=param", want: "image.jpg"},
		{url: "not-a-valid-url", want: "not-a-valid-url"},
		{url: "", want: "."},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := ExtractFilenameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("ExtractFilenameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
