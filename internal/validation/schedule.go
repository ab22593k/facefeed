package validation

import (
	"fmt"
	"strconv"
	"time"
)

// ParseSchedule converts a schedule time string to a Unix timestamp.
// It supports:
//   - Unix timestamps (numeric, seconds)
//   - ISO 8601 datetime strings (e.g. "2025-01-01T15:00:00Z")
//   - Relative durations ("30m", "2h", "1d", "7d")
func ParseSchedule(raw string) (int64, error) {
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
