package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTargeting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "targeting.json")
	if err := os.WriteFile(validFile, []byte(`{"age_min": 18, "age_max": 65}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	invalidFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFile, []byte(`{bad json}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	nonexistentFile := filepath.Join(tmpDir, "missing.json")

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		{
			name:  "empty string returns empty",
			input: "",
			check: func(t *testing.T, result string) {
				if result != "" {
					t.Errorf("got %q, want empty string", result)
				}
			},
		},
		{
			name:  "valid JSON string",
			input: `{"age_min": 18}`,
			check: func(t *testing.T, result string) {
				if result != `{"age_min": 18}` {
					t.Errorf("got %q, want %q", result, `{"age_min": 18}`)
				}
			},
		},
		{
			name:  "valid JSON with multiple fields",
			input: `{"age_min": 18, "age_max": 65, "genders": [1]}`,
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "age_max") {
					t.Errorf("result %q should contain age_max", result)
				}
			},
		},
		{
			name:    "invalid JSON returns error",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "malformed JSON returns error",
			input:   `{"key": broken}`,
			wantErr: true,
		},
		{
			name:  "@file with valid targeting JSON",
			input: "@" + validFile,
			check: func(t *testing.T, result string) {
				if result != `{"age_min": 18, "age_max": 65}` {
					t.Errorf("got %q, want %q", result, `{"age_min": 18, "age_max": 65}`)
				}
			},
		},
		{
			name:    "@file that does not exist returns error",
			input:   "@" + nonexistentFile,
			wantErr: true,
		},
		{
			name:    "@file with invalid JSON returns error",
			input:   "@" + invalidFile,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTargeting(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestParseTargeting_FileErrorMessage(t *testing.T) {
	t.Parallel()

	_, err := ParseTargeting("@/nonexistent/file/path.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not read targeting file") {
		t.Errorf("expected 'could not read targeting file' in error, got: %v", err)
	}
}

func TestParseTargeting_InvalidJSONErrorMessage(t *testing.T) {
	t.Parallel()

	_, err := ParseTargeting("{bad}")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid targeting JSON") {
		t.Errorf("expected 'invalid targeting JSON' in error, got: %v", err)
	}
}
