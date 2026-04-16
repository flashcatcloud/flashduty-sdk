package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

// ListAlertEventsInput contains parameters for listing alert events
type ListAlertEventsInput struct {
	AlertID string // Required: alert ID

	// Deprecated: /alert/event/list does not accept time-range filtering.
	StartTime int64
	// Deprecated: /alert/event/list does not accept time-range filtering.
	EndTime int64
	// Deprecated: /alert/event/list does not accept pagination.
	Limit int
	// Deprecated: /alert/event/list does not accept pagination.
	Page int
}

// ListAlertEventsOutput contains alert events
type ListAlertEventsOutput struct {
	AlertEvents []AlertEvent `json:"alert_events"`
}

// ListAlertEvents queries raw alert events
func (c *Client) ListAlertEvents(ctx context.Context, input *ListAlertEventsInput) (*ListAlertEventsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("alert event query input is required")
	}
	if input.AlertID == "" {
		return nil, fmt.Errorf("alert_id is required")
	}

	requestBody := map[string]any{
		"alert_id": input.AlertID,
	}

	resp, err := c.makeRequest(ctx, "POST", "/alert/event/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert events: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []AlertEvent `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	events := []AlertEvent{}
	if result.Data != nil {
		events = result.Data.Items
	}

	return &ListAlertEventsOutput{
		AlertEvents: events,
	}, nil
}
