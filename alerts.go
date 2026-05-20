package flashduty

import (
	"context"
	"fmt"
	"strings"
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

	result, err := postData[struct {
		Items []AlertEvent `json:"items"`
	}](c, ctx, "/alert/event/list", requestBody, "failed to list alert events")
	if err != nil {
		return nil, err
	}

	events := []AlertEvent{}
	if result != nil {
		events = result.Items
	}

	return &ListAlertEventsOutput{
		AlertEvents: events,
	}, nil
}

// ListAlertsInput contains parameters for listing alerts
type ListAlertsInput struct {
	StartTime      int64             // Required
	EndTime        int64             // Required
	AlertSeverity  string            // Optional
	IsActive       *bool             // Optional (pointer to distinguish unset from false)
	ChannelIDs     []int64           // Optional
	IntegrationIDs []int64           // Optional
	AlertKeys      []string          // Optional
	EverMuted      *bool             // Optional
	Title          string            // Optional
	Labels         map[string]string // Optional
	Limit          int               // Optional (default 20)
	Page           int               // Optional (default 1)
	OrderBy        string            // Optional
	Asc            bool              // Optional
	SearchAfterCtx string            // Optional: cursor for deep pagination
}

// ListAlertsOutput contains the result of listing alerts
type ListAlertsOutput struct {
	Alerts         []Alert `json:"alerts"`
	Total          int     `json:"total"`
	HasNextPage    bool    `json:"has_next_page"`
	SearchAfterCtx string  `json:"search_after_ctx,omitempty"`
}

// ListAlerts queries alerts by filters
func (c *Client) ListAlerts(ctx context.Context, input *ListAlertsInput) (*ListAlertsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("alert list input is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}

	requestBody := map[string]any{
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
		"limit":      limit,
		"p":          page,
	}
	if input.AlertSeverity != "" {
		requestBody["alert_severity"] = input.AlertSeverity
	}
	if input.IsActive != nil {
		requestBody["is_active"] = *input.IsActive
	}
	if len(input.ChannelIDs) > 0 {
		requestBody["channel_ids"] = input.ChannelIDs
	}
	if len(input.IntegrationIDs) > 0 {
		requestBody["integration_ids"] = input.IntegrationIDs
	}
	if len(input.AlertKeys) > 0 {
		requestBody["alert_keys"] = input.AlertKeys
	}
	if input.EverMuted != nil {
		requestBody["ever_muted"] = *input.EverMuted
	}
	if input.Title != "" {
		requestBody["title"] = input.Title
	}
	if len(input.Labels) > 0 {
		requestBody["labels"] = input.Labels
	}
	if input.OrderBy != "" {
		requestBody["orderby"] = input.OrderBy
	}
	if input.Asc {
		requestBody["asc"] = true
	}
	if input.SearchAfterCtx != "" {
		requestBody["search_after_ctx"] = input.SearchAfterCtx
	}

	result, err := postData[struct {
		Items          []Alert `json:"items"`
		Total          int     `json:"total"`
		HasNextPage    bool    `json:"has_next_page"`
		SearchAfterCtx string  `json:"search_after_ctx,omitempty"`
	}](c, ctx, "/alert/list", requestBody, "failed to list alerts")
	if err != nil {
		return nil, err
	}

	alerts := []Alert{}
	total := 0
	hasNextPage := false
	searchAfterCtx := ""
	if result != nil {
		alerts = result.Items
		total = result.Total
		hasNextPage = result.HasNextPage
		searchAfterCtx = result.SearchAfterCtx
	}

	return &ListAlertsOutput{
		Alerts:         alerts,
		Total:          total,
		HasNextPage:    hasNextPage,
		SearchAfterCtx: searchAfterCtx,
	}, nil
}

// GetAlertDetailInput contains parameters for getting alert detail
type GetAlertDetailInput struct {
	AlertID string // Required
}

// GetAlertDetailOutput contains full alert detail
type GetAlertDetailOutput struct {
	Alert Alert `json:"alert"`
}

// GetAlertDetail fetches detailed information for a single alert
func (c *Client) GetAlertDetail(ctx context.Context, input *GetAlertDetailInput) (*GetAlertDetailOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("alert detail input is required")
	}

	requestBody := map[string]any{
		"alert_id": input.AlertID,
	}

	alert, err := postOptionalData[Alert](c, ctx, "/alert/info", requestBody, "failed to get alert detail")
	if err != nil {
		return nil, err
	}

	if alert == nil {
		return nil, fmt.Errorf("alert not found: %s", input.AlertID)
	}

	return &GetAlertDetailOutput{Alert: *alert}, nil
}

// ListAlertsByIDs fetches alerts by their IDs
func (c *Client) ListAlertsByIDs(ctx context.Context, alertIDs []string) (*ListAlertsOutput, error) {
	requestBody := map[string]any{
		"alert_ids": alertIDs,
	}

	result, err := postData[struct {
		Items []Alert `json:"items"`
	}](c, ctx, "/alert/list-by-ids", requestBody, "failed to list alerts by IDs")
	if err != nil {
		return nil, err
	}

	alerts := []Alert{}
	if result != nil {
		alerts = result.Items
	}

	return &ListAlertsOutput{
		Alerts: alerts,
		Total:  len(alerts),
	}, nil
}

// MergeAlertsInput contains parameters for merging alerts into an incident
type MergeAlertsInput struct {
	AlertIDs   []string // Required
	IncidentID string   // Required
	Comment    string   // Optional
	Title      string   // Optional
}

// MergeAlertsToIncident merges alerts into an existing incident
func (c *Client) MergeAlertsToIncident(ctx context.Context, input *MergeAlertsInput) error {
	if input == nil {
		return fmt.Errorf("merge alerts input is required")
	}

	requestBody := map[string]any{
		"alert_ids":   input.AlertIDs,
		"incident_id": input.IncidentID,
	}
	if input.Comment != "" {
		requestBody["comment"] = input.Comment
	}
	if input.Title != "" {
		requestBody["title"] = input.Title
	}

	return postEmpty(c, ctx, "/alert/merge", requestBody, "failed to merge alerts to incident")
}

// GetAlertFeedInput contains parameters for getting alert feed/timeline
type GetAlertFeedInput struct {
	AlertID string   // Required
	Limit   int      // Optional (default 20)
	Page    int      // Optional (default 1)
	Asc     bool     // Optional
	Types   []string // Optional: filter by event type
}

// GetAlertFeedOutput contains alert feed events
type GetAlertFeedOutput struct {
	Items       []TimelineEvent `json:"items"`
	HasNextPage bool            `json:"has_next_page"`
}

// GetAlertFeed fetches the feed/timeline for an alert with enriched person names
func (c *Client) GetAlertFeed(ctx context.Context, input *GetAlertFeedInput) (*GetAlertFeedOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("alert feed input is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}

	requestBody := map[string]any{
		"alert_id": input.AlertID,
		"limit":    limit,
		"p":        page,
		"asc":      input.Asc,
	}
	if len(input.Types) > 0 {
		requestBody["types"] = input.Types
	}

	result, err := postData[struct {
		Items       []RawTimelineItem `json:"items"`
		HasNextPage bool              `json:"has_next_page"`
	}](c, ctx, "/alert/feed", requestBody, "failed to get alert feed")
	if err != nil {
		return nil, err
	}

	if result == nil || len(result.Items) == 0 {
		return &GetAlertFeedOutput{
			Items:       []TimelineEvent{},
			HasNextPage: false,
		}, nil
	}

	// Enrich with person names
	personIDs := collectTimelinePersonIDs(result.Items)
	personMap, err := c.fetchPersonInfos(ctx, personIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load person details for feed: %w", err)
	}

	enrichedItems := enrichTimelineItems(result.Items, personMap)

	return &GetAlertFeedOutput{
		Items:       enrichedItems,
		HasNextPage: result.HasNextPage,
	}, nil
}

// ListAlertEventsGlobalInput contains parameters for listing alert events globally
type ListAlertEventsGlobalInput struct {
	StartTime        int64    // Required
	EndTime          int64    // Required
	IntegrationTypes []string // Optional
	IntegrationIDs   []int64  // Optional
	ChannelIDs       []int64  // Optional
	Severities       []string // Optional: serialized as a comma-separated list
	OrderBy          string   // Optional
	Limit            int      // Optional (default 20)
	Page             int      // Optional (default 1)
	SearchAfterCtx   string   // Optional
}

// ListAlertEventsGlobalOutput contains global alert event results
type ListAlertEventsGlobalOutput struct {
	AlertEvents    []AlertEvent `json:"alert_events"`
	Total          int          `json:"total"`
	HasNextPage    bool         `json:"has_next_page"`
	SearchAfterCtx string       `json:"search_after_ctx,omitempty"`
}

// ListAlertEventsGlobal queries alert events globally across all alerts
func (c *Client) ListAlertEventsGlobal(ctx context.Context, input *ListAlertEventsGlobalInput) (*ListAlertEventsGlobalOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("alert event global query input is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}

	requestBody := map[string]any{
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
		"limit":      limit,
		"p":          page,
	}
	if len(input.IntegrationTypes) > 0 {
		requestBody["integration_types"] = input.IntegrationTypes
	}
	if len(input.IntegrationIDs) > 0 {
		requestBody["integration_ids"] = input.IntegrationIDs
	}
	if len(input.ChannelIDs) > 0 {
		requestBody["channel_ids"] = input.ChannelIDs
	}
	if len(input.Severities) > 0 {
		requestBody["severities"] = strings.Join(input.Severities, ",")
	}
	if input.OrderBy != "" {
		requestBody["orderby"] = input.OrderBy
	}
	if input.SearchAfterCtx != "" {
		requestBody["search_after_ctx"] = input.SearchAfterCtx
	}

	result, err := postData[struct {
		Items          []AlertEvent `json:"items"`
		Total          int          `json:"total"`
		HasNextPage    bool         `json:"has_next_page"`
		SearchAfterCtx string       `json:"search_after_ctx,omitempty"`
	}](c, ctx, "/alert-event/list", requestBody, "failed to list alert events")
	if err != nil {
		return nil, err
	}

	events := []AlertEvent{}
	total := 0
	hasNextPage := false
	searchAfterCtx := ""
	if result != nil {
		events = result.Items
		total = result.Total
		hasNextPage = result.HasNextPage
		searchAfterCtx = result.SearchAfterCtx
	}

	return &ListAlertEventsGlobalOutput{
		AlertEvents:    events,
		Total:          total,
		HasNextPage:    hasNextPage,
		SearchAfterCtx: searchAfterCtx,
	}, nil
}
