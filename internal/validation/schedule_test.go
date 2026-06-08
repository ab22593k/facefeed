package validation

import (
	"strings"
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, ts int64, err error)
	}{
		{
			name:  "empty string returns zero",
			input: "",
			check: func(t *testing.T, ts int64, err error) {
				if ts != 0 {
					t.Errorf("got ts %d, want 0", ts)
				}
			},
		},
		{
			name:  "valid unix timestamp",
			input: "1800000000",
			check: func(t *testing.T, ts int64, err error) {
				if ts != 1800000000 {
					t.Errorf("got ts %d, want 1800000000", ts)
				}
			},
		},
		{
			name:  "future unix timestamp",
			input: "2000000000",
			check: func(t *testing.T, ts int64, err error) {
				if ts != 2000000000 {
					t.Errorf("got ts %d, want 2000000000", ts)
				}
			},
		},
		{
			name:    "unix timestamp in the past returns error",
			input:   "1000000",
			wantErr: true,
		},
		{
			name:  "ISO 8601 RFC3339 format",
			input: "2099-12-31T23:59:00Z",
			check: func(t *testing.T, ts int64, err error) {
				expected, _ := time.Parse(time.RFC3339, "2099-12-31T23:59:00Z")
				if ts != expected.Unix() {
					t.Errorf("got ts %d, want %d", ts, expected.Unix())
				}
			},
		},
		{
			name:  "ISO 8601 without timezone",
			input: "2099-12-31T23:59:00",
			check: func(t *testing.T, ts int64, err error) {
				if ts <= 0 {
					t.Errorf("expected positive timestamp, got %d", ts)
				}
			},
		},
		{
			name:  "ISO 8601 with spaces",
			input: "2099-12-31 23:59:00",
			check: func(t *testing.T, ts int64, err error) {
				if ts <= 0 {
					t.Errorf("expected positive timestamp, got %d", ts)
				}
			},
		},
		{
			name:  "ISO 8601 date only",
			input: "2099-12-31",
			check: func(t *testing.T, ts int64, err error) {
				if ts <= 0 {
					t.Errorf("expected positive timestamp, got %d", ts)
				}
			},
		},
		{
			name:    "ISO 8601 in the past returns error",
			input:   "2020-01-01T00:00:00Z",
			wantErr: true,
		},
		{
			name:  "relative duration in minutes",
			input: "30m",
			check: func(t *testing.T, ts int64, err error) {
				expected := time.Now().Add(30 * time.Minute).Unix()
				diff := ts - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > 2 {
					t.Errorf("got ts %d, expected ~%d (diff %d)", ts, expected, diff)
				}
			},
		},
		{
			name:  "relative duration in hours",
			input: "2h",
			check: func(t *testing.T, ts int64, err error) {
				expected := time.Now().Add(2 * time.Hour).Unix()
				diff := ts - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > 2 {
					t.Errorf("got ts %d, expected ~%d (diff %d)", ts, expected, diff)
				}
			},
		},
		{
			name:  "relative duration in days",
			input: "1d",
			check: func(t *testing.T, ts int64, err error) {
				expected := time.Now().Add(24 * time.Hour).Unix()
				diff := ts - expected
				if diff < 0 {
					diff = -diff
				}
				if diff > 2 {
					t.Errorf("got ts %d, expected ~%d (diff %d)", ts, expected, diff)
				}
			},
		},
		{
			name:    "relative duration with zero value returns error",
			input:   "0d",
			wantErr: true,
		},
		{
			name:    "unsupported duration unit returns error",
			input:   "5x",
			wantErr: true,
		},
		{
			name:    "invalid text returns error",
			input:   "next week",
			wantErr: true,
		},
		{
			name:    "non-numeric prefix returns error",
			input:   "abc30m",
			wantErr: true,
		},
		{
			name:  "dynamic future timestamp passes validation",
			input: "1900000000",
			check: func(t *testing.T, ts int64, err error) {
				if ts != 1900000000 {
					t.Errorf("got ts %d, want 1900000000", ts)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := ParseSchedule(tt.input)

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
				tt.check(t, ts, err)
			}
		})
	}
}

func TestParseSchedule_PastTimestamp(t *testing.T) {
	t.Parallel()

	// A timestamp that's definitely in the past.
	_, err := ParseSchedule("1000000")
	if err == nil {
		t.Fatal("expected error for past timestamp, got nil")
	}
	if !strings.Contains(err.Error(), "is in the past") {
		t.Errorf("expected 'is in the past' in error, got: %v", err)
	}
}

func TestParseSchedule_PastISO8601(t *testing.T) {
	t.Parallel()

	// An ISO 8601 date that's definitely in the past.
	_, err := ParseSchedule("2020-06-01T12:00:00Z")
	if err == nil {
		t.Fatal("expected error for past ISO 8601, got nil")
	}
	if !strings.Contains(err.Error(), "is in the past") {
		t.Errorf("expected 'is in the past' in error, got: %v", err)
	}
}
