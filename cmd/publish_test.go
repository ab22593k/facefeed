package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTargets(t *testing.T) {
	t.Parallel()

	// Set up a temporary config file for config-path tests.
	configDir := t.TempDir()
	validConfigPath := filepath.Join(configDir, "targets.json")
	validConfigContent := `[
		{"id": "config_page_1", "message": "Hello from config"},
		{"id": "config_page_2", "message": ""}
	]`
	if err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	nonexistentPath := filepath.Join(configDir, "nonexistent.json")
	invalidConfigPath := filepath.Join(configDir, "invalid.json")
	if err := os.WriteFile(invalidConfigPath, []byte(`{bad json}`), 0644); err != nil {
		t.Fatalf("failed to write invalid config file: %v", err)
	}

	tests := []struct {
		name       string
		pageID     string
		groups     string
		configPath string
		wantLen    int
		wantTarget func(t *testing.T, targets []PublishTarget)
		wantErr    bool
	}{
		{
			name:       "empty inputs returns empty targets",
			pageID:     "",
			groups:     "",
			configPath: "",
			wantLen:    0,
			wantTarget: nil,
			wantErr:    false,
		},
		{
			name:       "page ID only",
			pageID:     "page_123",
			groups:     "",
			configPath: "",
			wantLen:    1,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if targets[0].ID != "page_123" || targets[0].Message != "" {
					t.Errorf("got %+v, want {ID: page_123, Message: }", targets[0])
				}
			},
			wantErr: false,
		},
		{
			name:       "single group",
			pageID:     "",
			groups:     "group_456",
			configPath: "",
			wantLen:    1,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if targets[0].ID != "group_456" {
					t.Errorf("got ID %q, want %q", targets[0].ID, "group_456")
				}
			},
			wantErr: false,
		},
		{
			name:       "multiple groups",
			pageID:     "",
			groups:     "g1,g2,g3",
			configPath: "",
			wantLen:    3,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				ids := make([]string, len(targets))
				for i, t := range targets {
					ids[i] = t.ID
				}
				expected := []string{"g1", "g2", "g3"}
				for i, id := range ids {
					if id != expected[i] {
						t.Errorf("target[%d].ID = %q, want %q", i, id, expected[i])
					}
				}
			},
			wantErr: false,
		},
		{
			name:       "groups with whitespace",
			pageID:     "",
			groups:     " g1 , g2 , g3 ",
			configPath: "",
			wantLen:    3,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if targets[0].ID != "g1" {
					t.Errorf("target[0].ID = %q, want %q", targets[0].ID, "g1")
				}
			},
			wantErr: false,
		},
		{
			name:       "empty group IDs are skipped",
			pageID:     "",
			groups:     "g1,,g2, ,g3",
			configPath: "",
			wantLen:    3,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if len(targets) != 3 {
					t.Fatalf("got %d targets, want 3", len(targets))
				}
			},
			wantErr: false,
		},
		{
			name:       "page ID and groups combined",
			pageID:     "page_123",
			groups:     "g1,g2",
			configPath: "",
			wantLen:    3,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if targets[0].ID != "page_123" {
					t.Errorf("target[0].ID = %q, want %q", targets[0].ID, "page_123")
				}
				if targets[1].ID != "g1" {
					t.Errorf("target[1].ID = %q, want %q", targets[1].ID, "g1")
				}
				if targets[2].ID != "g2" {
					t.Errorf("target[2].ID = %q, want %q", targets[2].ID, "g2")
				}
			},
			wantErr: false,
		},
		{
			name:       "valid config file",
			pageID:     "",
			groups:     "",
			configPath: validConfigPath,
			wantLen:    2,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if targets[0].ID != "config_page_1" || targets[0].Message != "Hello from config" {
					t.Errorf("target[0] = %+v, want {ID: config_page_1, Message: Hello from config}", targets[0])
				}
				if targets[1].ID != "config_page_2" || targets[1].Message != "" {
					t.Errorf("target[1] = %+v, want {ID: config_page_2, Message: }", targets[1])
				}
			},
			wantErr: false,
		},
		{
			name:       "config file takes precedence (pageID and groups ignored)",
			pageID:     "should_be_ignored",
			groups:     "should_be_ignored",
			configPath: validConfigPath,
			wantLen:    2,
			wantTarget: func(t *testing.T, targets []PublishTarget) {
				if targets[0].ID == "should_be_ignored" {
					t.Error("config did not take precedence; page ID was used")
				}
			},
			wantErr: false,
		},
		{
			name:       "nonexistent config file returns error",
			pageID:     "",
			groups:     "",
			configPath: nonexistentPath,
			wantLen:    0,
			wantTarget: nil,
			wantErr:    true,
		},
		{
			name:       "invalid JSON config returns error",
			pageID:     "",
			groups:     "",
			configPath: invalidConfigPath,
			wantLen:    0,
			wantTarget: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := resolveTargets(tt.pageID, tt.groups, tt.configPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(targets) != tt.wantLen {
				t.Fatalf("got %d targets, want %d", len(targets), tt.wantLen)
			}

			if tt.wantTarget != nil {
				tt.wantTarget(t, targets)
			}
		})
	}
}

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
		check      func(t *testing.T, images []ImageInput, result *ValidationResult)
	}{
		{
			name:       "empty message and no images is invalid",
			imagePaths: nil,
			message:    "",
			wantValid:  false,
			wantImages: 0,
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
				if len(result.Errors) == 0 {
					t.Error("expected error about missing message/images, got none")
				}
			},
		},
		{
			name:       "message only with no images is valid",
			imagePaths: nil,
			message:    "Hello, world!",
			wantValid:  true,
			wantImages: 0,
		},
		{
			name:       "URL image is valid",
			imagePaths: []string{"https://example.com/image.jpg"},
			message:    "Check this out",
			wantValid:  true,
			wantImages: 1,
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
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
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
				for i, img := range images {
					if img.Type != "url" {
						t.Errorf("images[%d].Type = %q, want %q", i, img.Type, "url")
					}
				}
			},
		},
		{
			name:       "valid local file",
			imagePaths: []string{validJPEG},
			message:    "My photo",
			wantValid:  true,
			wantImages: 1,
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
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
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
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
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
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
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
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
			check: func(t *testing.T, images []ImageInput, result *ValidationResult) {
				if images[0].Type != "url" {
					t.Errorf("images[0].Type = %q, want %q", images[0].Type, "url")
				}
				if images[1].Type != "file" {
					t.Errorf("images[1].Type = %q, want %q", images[1].Type, "file")
				}
			},
		},
		{
			name:       "multiple images without message is still valid (images suffice)",
			imagePaths: []string{validJPEG, validPNG},
			message:    "",
			wantValid:  true,
			wantImages: 2,
		},
		{
			name:       "URL with no message is valid (images suffice)",
			imagePaths: []string{"https://example.com/photo.jpg"},
			message:    "",
			wantValid:  true,
			wantImages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, result := validateInputs(tt.imagePaths, tt.message)

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
			got := detectInputType(tt.input)
			if got != tt.want {
				t.Errorf("detectInputType(%q) = %q, want %q", tt.input, got, tt.want)
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
			got := extractFilenameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractFilenameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
