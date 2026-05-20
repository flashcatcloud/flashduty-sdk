package flashduty

import (
	"context"
	"fmt"
)

// InsightQueryInput contains common parameters for all /insight/* endpoints
type InsightQueryInput struct {
	StartTime          int64             // Required: Unix seconds
	EndTime            int64             // Required: Unix seconds
	TeamIDs            []int64           // Filter by teams (max 100)
	ChannelIDs         []int64           // Filter by channels (max 100)
	ResponderIDs       []int64           // Filter by responders (max 100)
	Severities         []string          // Filter: Critical, Warning, Info
	SplitHours         bool              // Split metrics into work/sleep/off buckets
	AggregateUnit      string            // day, week, or month for time-series
	TimeZone           string            // IANA timezone (e.g., Asia/Shanghai)
	Labels             map[string]string // Exact-match label filters
	Fields             map[string]string // Exact-match field filters
	SecondsToAckFrom   int               // Range filter on acknowledgment time (lower bound)
	SecondsToAckTo     int               // Range filter on acknowledgment time (upper bound)
	SecondsToCloseFrom int               // Range filter on resolution time (lower bound)
	SecondsToCloseTo   int               // Range filter on resolution time (upper bound)
}

// buildRequestBody constructs the common request body from InsightQueryInput
func (input *InsightQueryInput) buildRequestBody() map[string]any {
	if input == nil {
		return map[string]any{}
	}
	body := map[string]any{
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
	}
	if len(input.TeamIDs) > 0 {
		body["team_ids"] = input.TeamIDs
	}
	if len(input.ChannelIDs) > 0 {
		body["channel_ids"] = input.ChannelIDs
	}
	if len(input.ResponderIDs) > 0 {
		body["responder_ids"] = input.ResponderIDs
	}
	if len(input.Severities) > 0 {
		body["severities"] = input.Severities
	}
	if input.SplitHours {
		body["split_hours"] = true
	}
	if input.AggregateUnit != "" {
		body["aggregate_unit"] = input.AggregateUnit
	}
	if input.TimeZone != "" {
		body["time_zone"] = input.TimeZone
	}
	if len(input.Labels) > 0 {
		body["labels"] = input.Labels
	}
	if len(input.Fields) > 0 {
		body["fields"] = input.Fields
	}
	if input.SecondsToAckFrom > 0 {
		body["seconds_to_ack_from"] = input.SecondsToAckFrom
	}
	if input.SecondsToAckTo > 0 {
		body["seconds_to_ack_to"] = input.SecondsToAckTo
	}
	if input.SecondsToCloseFrom > 0 {
		body["seconds_to_close_from"] = input.SecondsToCloseFrom
	}
	if input.SecondsToCloseTo > 0 {
		body["seconds_to_close_to"] = input.SecondsToCloseTo
	}
	return body
}

// QueryInsightByTeamOutput contains team-level insight metrics
type QueryInsightByTeamOutput struct {
	Items []DimensionInsightItem `json:"items"`
}

// QueryInsightByTeam queries pre-aggregated metrics grouped by team
func (c *Client) QueryInsightByTeam(ctx context.Context, input *InsightQueryInput) (*QueryInsightByTeamOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query input is required")
	}

	result, err := postData[struct {
		Items []DimensionInsightItem `json:"items"`
	}](c, ctx, "/insight/team", input.buildRequestBody(), "failed to query team insights")
	if err != nil {
		return nil, err
	}

	items := []DimensionInsightItem{}
	if result != nil {
		items = result.Items
	}

	return &QueryInsightByTeamOutput{Items: items}, nil
}

// QueryInsightByResponderOutput contains per-responder insight metrics
type QueryInsightByResponderOutput struct {
	Items []ResponderInsightItem `json:"items"`
}

// QueryInsightByResponder queries pre-aggregated metrics grouped by responder
func (c *Client) QueryInsightByResponder(ctx context.Context, input *InsightQueryInput) (*QueryInsightByResponderOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query input is required")
	}

	result, err := postData[struct {
		Items []ResponderInsightItem `json:"items"`
	}](c, ctx, "/insight/responder", input.buildRequestBody(), "failed to query responder insights")
	if err != nil {
		return nil, err
	}

	items := []ResponderInsightItem{}
	if result != nil {
		items = result.Items
	}

	return &QueryInsightByResponderOutput{Items: items}, nil
}

// QueryInsightByChannelOutput contains per-channel insight metrics
type QueryInsightByChannelOutput struct {
	Items []DimensionInsightItem `json:"items"`
}

// QueryInsightByChannel queries pre-aggregated metrics grouped by channel
func (c *Client) QueryInsightByChannel(ctx context.Context, input *InsightQueryInput) (*QueryInsightByChannelOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query input is required")
	}

	result, err := postData[struct {
		Items []DimensionInsightItem `json:"items"`
	}](c, ctx, "/insight/channel", input.buildRequestBody(), "failed to query channel insights")
	if err != nil {
		return nil, err
	}

	items := []DimensionInsightItem{}
	if result != nil {
		items = result.Items
	}

	return &QueryInsightByChannelOutput{Items: items}, nil
}

// QueryInsightAlertTopKInput contains parameters for querying top alert sources
type QueryInsightAlertTopKInput struct {
	InsightQueryInput
	Label   string // Required: "check" or "resource"
	K       int    // Top K results (1-100, default 20)
	OrderBy string // "total_alert_cnt" or "total_alert_event_cnt"
	Asc     bool   // Sort ascending when true
}

// QueryInsightAlertTopKOutput contains top-K alert sources
type QueryInsightAlertTopKOutput struct {
	Items []InsightAlertByLabelItem `json:"items"`
}

// QueryInsightAlertTopK queries top alert sources grouped by a label
func (c *Client) QueryInsightAlertTopK(ctx context.Context, input *QueryInsightAlertTopKInput) (*QueryInsightAlertTopKOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query input is required")
	}

	body := input.InsightQueryInput.buildRequestBody()
	if input.Label != "" {
		body["label"] = input.Label
	}
	k := input.K
	if k <= 0 {
		k = defaultQueryLimit
	}
	body["k"] = k
	if input.OrderBy != "" {
		body["orderby"] = input.OrderBy
	}
	if input.Asc {
		body["asc"] = true
	}

	result, err := postData[struct {
		Items []InsightAlertByLabelItem `json:"items"`
	}](c, ctx, "/insight/alert/topk-by-label", body, "failed to query alert top-K")
	if err != nil {
		return nil, err
	}

	items := []InsightAlertByLabelItem{}
	if result != nil {
		items = result.Items
	}

	return &QueryInsightAlertTopKOutput{Items: items}, nil
}

// QueryInsightIncidentListInput contains parameters for querying incidents with metrics
type QueryInsightIncidentListInput struct {
	InsightQueryInput
	Limit          int    // Max results (default 20)
	Page           int    // Page number (default 1)
	SearchAfterCtx string // Cursor for the next page when returned by the API
	OrderBy        string // Deprecated: the API does not support orderby for this endpoint
}

// QueryInsightIncidentListOutput contains incidents with performance metrics
type QueryInsightIncidentListOutput struct {
	Items          []InsightIncidentItem `json:"items"`
	Total          int                   `json:"total"`
	HasNextPage    bool                  `json:"has_next_page"`
	SearchAfterCtx string                `json:"search_after_ctx,omitempty"`
}

// QueryInsightIncidentList queries incidents with attached performance metrics
func (c *Client) QueryInsightIncidentList(ctx context.Context, input *QueryInsightIncidentListInput) (*QueryInsightIncidentListOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("query input is required")
	}

	body := input.InsightQueryInput.buildRequestBody()

	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}
	body["limit"] = limit
	body["p"] = page

	if input.SearchAfterCtx != "" {
		body["search_after_ctx"] = input.SearchAfterCtx
	}

	result, err := postData[struct {
		Items          []InsightIncidentItem `json:"items"`
		Total          int                   `json:"total"`
		HasNextPage    bool                  `json:"has_next_page"`
		SearchAfterCtx string                `json:"search_after_ctx,omitempty"`
	}](c, ctx, "/insight/incident/list", body, "failed to query insight incidents")
	if err != nil {
		return nil, err
	}

	items := []InsightIncidentItem{}
	total := 0
	hasNextPage := false
	searchAfterCtx := ""
	if result != nil {
		items = result.Items
		total = result.Total
		hasNextPage = result.HasNextPage
		searchAfterCtx = result.SearchAfterCtx
	}

	return &QueryInsightIncidentListOutput{
		Items:          items,
		Total:          total,
		HasNextPage:    hasNextPage,
		SearchAfterCtx: searchAfterCtx,
	}, nil
}
