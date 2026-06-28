package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HandleResponse parses a Facebook API response and extracts the post/media ID.
func HandleResponse(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %s, body: %s", resp.Status, string(body))
	}

	var res map[string]any
	_ = json.Unmarshal(body, &res)

	if id, ok := res["id"].(string); ok {
		return id, nil
	}

	return string(body), nil
}

// addScheduleParams adds published=false and scheduled_publish_time to the URL values
// when a schedule time is specified.
func addScheduleParams(data map[string]string, scheduleUnix int64) {
	if scheduleUnix > 0 {
		data["published"] = "false"
		data["scheduled_publish_time"] = fmt.Sprintf("%d", scheduleUnix)
	}
}
