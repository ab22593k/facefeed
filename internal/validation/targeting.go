package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParseTargeting accepts either a JSON string or a file path prefixed with "@filename"
// and returns the validated targeting JSON string (or empty if not provided).
func ParseTargeting(raw string) (string, error) {
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
