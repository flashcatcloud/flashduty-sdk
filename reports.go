package flashduty

import (
	"context"
	"fmt"
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

	result, err := postData[struct {
		Items []NotificationTrendPoint `json:"items"`
	}](c, ctx, "/report/oncall/notifications", requestBody, "failed to query notification trend")
	if err != nil {
		return nil, err
	}

	dataPoints := []NotificationTrendPoint{}
	if result != nil {
		dataPoints = result.Items
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

	result, err := postData[struct {
		Items []ChangeTrendPoint `json:"items"`
	}](c, ctx, "/report/oncall/changes", requestBody, "failed to query change trend")
	if err != nil {
		return nil, err
	}

	dataPoints := []ChangeTrendPoint{}
	if result != nil {
		dataPoints = result.Items
	}

	return &QueryChangeTrendOutput{DataPoints: dataPoints}, nil
}
