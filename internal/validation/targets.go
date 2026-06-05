package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"facefeed/domain"
)

// ResolveTargets resolves publish targets from page ID, groups flag, or config file.
// Config file takes precedence over page ID and groups.
func ResolveTargets(pageID, groups, configPath string) ([]domain.PublishTarget, error) {
	var targets []domain.PublishTarget

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
		targets = append(targets, domain.PublishTarget{ID: pageID})
	}

	if groups != "" {
		groupIDs := strings.Split(groups, ",")
		for _, id := range groupIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				targets = append(targets, domain.PublishTarget{ID: id})
			}
		}
	}

	return targets, nil
}
