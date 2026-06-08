package presentation

import (
	"fmt"

	"facefeed/domain"
	theme "facefeed/internal"
)

// DisplayPublishResult shows the result of a content publish operation.
func DisplayPublishResult(contentType, postID string, err error) {
	if err != nil {
		theme.Error(fmt.Sprintf("%s publish failed: %v", contentType, err))
		return
	}
	theme.Success(fmt.Sprintf("%s published! ID: %s", contentType, postID))
}

// DisplayInsights displays insight metrics data.
func DisplayInsights(data []domain.InsightData, err error) {
	if err != nil {
		theme.Error(err.Error())
		return
	}

	if len(data) == 0 {
		theme.Info("Status", "No insight data available for the requested metrics.")
		return
	}

	for _, metric := range data {
		fmt.Printf("\n")
		theme.Info("Metric", metric.Name)
		theme.Info("Period", metric.Period)

		for _, v := range metric.Values {
			valStr := formatInsightValue(v.Value)
			timeStr := statusUnknown
			if v.EndTime != "" {
				timeStr = v.EndTime
			}
			theme.Info("Value", fmt.Sprintf("%s (as of %s)", valStr, timeStr))
		}
	}

	theme.Info("Total", fmt.Sprintf("%d metric(s) returned", len(data)))
}

// formatInsightValue converts an insight value to a readable string.
func formatInsightValue(v interface{}) string {
	if v == nil {
		return "0"
	}
	switch val := v.(type) {
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case string:
		if val == "" {
			return "0"
		}
		return val
	case int64:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
