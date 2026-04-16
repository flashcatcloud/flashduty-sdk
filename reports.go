package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

// QueryNotificationTrendInput contains parameters for querying notification trends
type QueryNotificationTrendInput struct {
	ChannelIDs []int64 // Optional: filter by channels
	Step       string  // Required: day, week, or month
	StartTime  int64   // Required: Unix seconds
	EndTime    int64   // Required: Unix seconds
}

// QueryNotificationTrendOutput contains notification volume trend data
type QueryNotificationTrendOutput struct {
	DataPoints []NotificationTrendPoint `json:"data_points"`
}

// NotificationTrendPoint preserves the per-channel notification counters for each time bucket.
type NotificationTrendPoint struct {
	Timestamp  int64 `json:"ts"`
	SMSCount   int   `json:"sms_cnt"`
	VoiceCount int   `json:"voice_cnt"`
	EmailCount int   `json:"email_cnt"`
}

// QueryNotificationTrend queries notification volume trends over time
func (c *Client) QueryNotificationTrend(ctx context.Context, input *QueryNotificationTrendInput) (*QueryNotificationTrendOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query notification trend input is required")
	}
	if input.Step == "" || input.StartTime <= 0 || input.EndTime <= 0 {
		return nil, fmt.Errorf("step, start_time, and end_time are required")
	}

	requestBody := map[string]any{
		"step":       input.Step,
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
	}
	if len(input.ChannelIDs) > 0 {
		requestBody["channel_ids"] = input.ChannelIDs
	}

	resp, err := c.makeRequest(ctx, "POST", "/report/oncall/notifications", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification trend: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []NotificationTrendPoint `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	dataPoints := []NotificationTrendPoint{}
	if result.Data != nil {
		dataPoints = result.Data.Items
	}

	return &QueryNotificationTrendOutput{DataPoints: dataPoints}, nil
}

// QueryChangeTrendInput contains parameters for querying change volume trends
type QueryChangeTrendInput struct {
	Step      string // Required: day, week, or month
	StartTime int64  // Required: Unix seconds
	EndTime   int64  // Required: Unix seconds
}

// QueryChangeTrendOutput contains change volume trend data
type QueryChangeTrendOutput struct {
	DataPoints []ChangeTrendPoint `json:"data_points"`
}

// ChangeTrendPoint preserves the change counters for each time bucket.
type ChangeTrendPoint struct {
	Timestamp        int64 `json:"ts"`
	ChangeCount      int   `json:"change_cnt"`
	ChangeEventCount int   `json:"change_event_cnt"`
}

// QueryChangeTrend queries change volume trends over time
func (c *Client) QueryChangeTrend(ctx context.Context, input *QueryChangeTrendInput) (*QueryChangeTrendOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query change trend input is required")
	}
	if input.Step == "" || input.StartTime <= 0 || input.EndTime <= 0 {
		return nil, fmt.Errorf("step, start_time, and end_time are required")
	}

	requestBody := map[string]any{
		"step":       input.Step,
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
	}

	resp, err := c.makeRequest(ctx, "POST", "/report/oncall/changes", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query change trend: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []ChangeTrendPoint `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	dataPoints := []ChangeTrendPoint{}
	if result.Data != nil {
		dataPoints = result.Data.Items
	}

	return &QueryChangeTrendOutput{DataPoints: dataPoints}, nil
}
