package validation

import (
	"os"
	"path/filepath"
	"testing"

	"facefeed/domain"
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
		wantTarget func(t *testing.T, targets []domain.PublishTarget)
		wantErr    bool
	}{
		{
			name:    "empty inputs returns empty targets",
			wantLen: 0,
		},
		{
			name:    "page ID only",
			pageID:  "page_123",
			wantLen: 1,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
				if targets[0].ID != "page_123" || targets[0].Message != "" {
					t.Errorf("got %+v, want {ID: page_123, Message: }", targets[0])
				}
			},
		},
		{
			name:    "single group",
			groups:  "group_456",
			wantLen: 1,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
				if targets[0].ID != "group_456" {
					t.Errorf("got ID %q, want %q", targets[0].ID, "group_456")
				}
			},
		},
		{
			name:    "multiple groups",
			groups:  "g1,g2,g3",
			wantLen: 3,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
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
		},
		{
			name:    "groups with whitespace",
			groups:  " g1 , g2 , g3 ",
			wantLen: 3,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
				if targets[0].ID != "g1" {
					t.Errorf("target[0].ID = %q, want %q", targets[0].ID, "g1")
				}
			},
		},
		{
			name:    "empty group IDs are skipped",
			groups:  "g1,,g2, ,g3",
			wantLen: 3,
		},
		{
			name:    "page ID and groups combined",
			pageID:  "page_123",
			groups:  "g1,g2",
			wantLen: 3,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
				if targets[0].ID != "page_123" {
					t.Errorf("target[0].ID = %q, want %q", targets[0].ID, "page_123")
				}
				if targets[1].ID != "g1" {
					t.Errorf("target[1].ID = %q, want %q", targets[1].ID, "g1")
				}
			},
		},
		{
			name:       "valid config file",
			configPath: validConfigPath,
			wantLen:    2,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
				if targets[0].ID != "config_page_1" || targets[0].Message != "Hello from config" {
					t.Errorf("target[0] = %+v, want {ID: config_page_1, Message: Hello from config}", targets[0])
				}
			},
		},
		{
			name:       "config file takes precedence",
			pageID:     "should_be_ignored",
			groups:     "should_be_ignored",
			configPath: validConfigPath,
			wantLen:    2,
			wantTarget: func(t *testing.T, targets []domain.PublishTarget) {
				if len(targets) > 0 && targets[0].ID == "should_be_ignored" {
					t.Error("config did not take precedence; page ID was used")
				}
			},
		},
		{
			name:       "nonexistent config file returns error",
			configPath: nonexistentPath,
			wantErr:    true,
		},
		{
			name:       "invalid JSON config returns error",
			configPath: invalidConfigPath,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := ResolveTargets(tt.pageID, tt.groups, tt.configPath)

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
