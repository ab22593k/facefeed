package validation

import (
	"errors"
	"strings"
	"testing"

	"facefeed/domain"
	"facefeed/internal/testutil"
)

func TestPrintDryRunSummary(t *testing.T) {
	tests := []struct {
		name    string
		result  *domain.ValidationResult
		targets []domain.PublishTarget
		want    []string
	}{
		{
			name: "valid with no images or targets",
			result: &domain.ValidationResult{
				Valid:  true,
				Images: nil,
			},
			targets: nil,
			want:    []string{"DRY RUN VALIDATION", "Valid", "true"},
		},
		{
			name: "valid with targets",
			result: &domain.ValidationResult{
				Valid:  true,
				Images: nil,
			},
			targets: []domain.PublishTarget{
				{ID: "page_123"},
				{ID: "group_456", Message: "Custom message"},
			},
			want: []string{"page_123", "group_456", "Default", "Custom"},
		},
		{
			name: "with images and warnings",
			result: &domain.ValidationResult{
				Valid:     true,
				TotalSize: 5 * 1024 * 1024,
				Images: []domain.ImageInput{
					{Path: "/path/to/photo.jpg", Type: "file", Size: 5 * 1024 * 1024, Filename: "photo.jpg"},
					{Path: "https://example.com/img.png", Type: "url", Filename: "img.png"},
				},
				Warnings: []string{"Large file detected"},
			},
			targets: []domain.PublishTarget{{ID: "page_123"}},
			want:    []string{"photo.jpg", "img.png", "5.00 MB", "Large file detected", "Warnings"},
		},
		{
			name: "with errors",
			result: &domain.ValidationResult{
				Valid:  false,
				Images: nil,
				Errors: []string{"File not found: missing.jpg"},
			},
			targets: []domain.PublishTarget{{ID: "page_123"}},
			want:    []string{"false", "Errors:", "File not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := testutil.CaptureStdout(func() {
				PrintDryRunSummary(tt.result, tt.targets)
			})

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestPrintBatchResultsSummary(t *testing.T) {
	tests := []struct {
		name    string
		results []domain.TargetResult
		want    []string
	}{
		{
			name:    "empty results",
			results: nil,
			want:    []string{"BATCH RESULTS SUMMARY"},
		},
		{
			name: "all images uploaded successfully",
			results: []domain.TargetResult{
				{
					TargetID: "page_123",
					Results: []domain.UploadResult{
						{Filename: "a.jpg", Success: true},
						{Filename: "b.jpg", Success: true},
					},
				},
			},
			want: []string{"page_123", "All 2 images uploaded"},
		},
		{
			name: "partial image upload success",
			results: []domain.TargetResult{
				{
					TargetID: "page_123",
					Results: []domain.UploadResult{
						{Filename: "a.jpg", Success: true},
						{Filename: "b.jpg", Success: false, Error: errors.New("upload failed")},
					},
				},
			},
			want: []string{"page_123", "1/2 images uploaded"},
		},
		{
			name: "text post published successfully",
			results: []domain.TargetResult{
				{
					TargetID: "page_123",
					PostID:   "post_456",
				},
			},
			want: []string{"page_123", "Published successfully", "post_456"},
		},
		{
			name: "text post failed",
			results: []domain.TargetResult{
				{
					TargetID: "page_123",
					Error:    errors.New("API error"),
				},
			},
			want: []string{"page_123", "Failed", "API error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := testutil.CaptureStdout(func() {
				PrintBatchResultsSummary(tt.results)
			})

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}
