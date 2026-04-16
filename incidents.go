package flashduty

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
)

const defaultQueryLimit = 20

// ListIncidentsInput contains parameters for listing incidents
type ListIncidentsInput struct {
	IncidentIDs   []string // Direct lookup by IDs (if set, other filters are ignored)
	Progress      string   // Filter: Triggered, Processing, Closed (comma-separated for multiple)
	Severity      string   // Filter: Info, Warning, Critical
	ChannelID     int64    // Filter by collaboration space ID
	StartTime     int64    // Unix timestamp (seconds), required if no IncidentIDs
	EndTime       int64    // Unix timestamp (seconds), required if no IncidentIDs
	Title         string   // Keyword search in incident title
	Limit         int      // Max results (default 20, max 100)
	Page          int      // Page number (default 1)
	IncludeAlerts bool     // Whether to include alerts preview (default true)
}

// ListIncidentsOutput contains the result of listing incidents
type ListIncidentsOutput struct {
	Incidents []EnrichedIncident `json:"incidents"`
	Total     int                `json:"total"`
}

// ListIncidents queries incidents by IDs or filters, returns enriched data with names
func (c *Client) ListIncidents(ctx context.Context, input *ListIncidentsInput) (*ListIncidentsOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}

	var rawIncidents []RawIncident
	var err error

	if len(input.IncidentIDs) > 0 {
		rawIncidents, err = c.fetchIncidentsByIDs(ctx, input.IncidentIDs)
	} else {
		if input.StartTime == 0 || input.EndTime == 0 {
			return nil, fmt.Errorf("both start_time and end_time are required for time-based queries")
		}
		page := input.Page
		if page <= 0 {
			page = 1
		}
		rawIncidents, err = c.fetchIncidentsByFilters(ctx, input.Progress, input.Severity, input.ChannelID, input.StartTime, input.EndTime, input.Title, limit, page)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve incidents: %w", err)
	}

	if len(rawIncidents) == 0 {
		return &ListIncidentsOutput{
			Incidents: []EnrichedIncident{},
			Total:     0,
		}, nil
	}

	enrichedIncidents, err := c.enrichIncidents(ctx, rawIncidents)
	if err != nil {
		return nil, fmt.Errorf("unable to load additional incident details: %w", err)
	}

	// Fetch alerts concurrently if requested
	if input.IncludeAlerts && len(enrichedIncidents) > 0 {
		g, gctx := errgroup.WithContext(ctx)
		for i := range enrichedIncidents {
			incidentID := enrichedIncidents[i].IncidentID
			g.Go(func() error {
				alerts, total, fetchErr := c.fetchIncidentAlerts(gctx, incidentID, defaultQueryLimit)
				if fetchErr != nil {
					return fetchErr
				}
				enrichedIncidents[i].AlertsPreview = alerts
				enrichedIncidents[i].AlertsTotal = total
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, fmt.Errorf("unable to retrieve alerts: %w", err)
		}
	}

	return &ListIncidentsOutput{
		Incidents: enrichedIncidents,
		Total:     len(enrichedIncidents),
	}, nil
}

// IncidentTimelineOutput contains timeline data for a single incident
type IncidentTimelineOutput struct {
	IncidentID string          `json:"incident_id"`
	Timeline   []TimelineEvent `json:"timeline"`
	Total      int             `json:"total"`
}

// GetIncidentTimelines fetches timelines for one or more incidents concurrently
func (c *Client) GetIncidentTimelines(ctx context.Context, incidentIDs []string) ([]IncidentTimelineOutput, error) {
	if len(incidentIDs) == 0 {
		return nil, fmt.Errorf("incident_ids must contain at least one valid ID")
	}

	type timelineResult struct {
		IncidentID string
		Items      []RawTimelineItem
	}
	results := make([]timelineResult, len(incidentIDs))

	g, gctx := errgroup.WithContext(ctx)
	for i, id := range incidentIDs {
		g.Go(func() error {
			items, err := c.fetchIncidentTimeline(gctx, id)
			if err != nil {
				return err
			}
			results[i] = timelineResult{IncidentID: id, Items: items}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("unable to retrieve timeline: %w", err)
	}

	// Collect all person IDs from all timelines
	allPersonIDs := make([]int64, 0)
	for _, r := range results {
		allPersonIDs = append(allPersonIDs, collectTimelinePersonIDs(r.Items)...)
	}

	// Batch fetch person info
	personMap, err := c.fetchPersonInfos(ctx, allPersonIDs)
	if err != nil {
		return nil, fmt.Errorf("unable to load person details: %w", err)
	}

	// Build enriched response
	output := make([]IncidentTimelineOutput, 0, len(results))
	for _, r := range results {
		enrichedEvents := enrichTimelineItems(r.Items, personMap)
		output = append(output, IncidentTimelineOutput{
			IncidentID: r.IncidentID,
			Timeline:   enrichedEvents,
			Total:      len(enrichedEvents),
		})
	}

	return output, nil
}

// IncidentAlertsOutput contains alerts data for a single incident
type IncidentAlertsOutput struct {
	IncidentID string         `json:"incident_id"`
	Alerts     []AlertPreview `json:"alerts"`
	Total      int            `json:"total"`
}

// ListIncidentAlerts fetches alerts for one or more incidents concurrently
func (c *Client) ListIncidentAlerts(ctx context.Context, incidentIDs []string, limit int) ([]IncidentAlertsOutput, error) {
	if len(incidentIDs) == 0 {
		return nil, fmt.Errorf("incident_ids must contain at least one valid ID")
	}

	if limit <= 0 {
		limit = defaultQueryLimit
	}

	results := make([]IncidentAlertsOutput, len(incidentIDs))

	g, gctx := errgroup.WithContext(ctx)
	for i, id := range incidentIDs {
		g.Go(func() error {
			alerts, total, err := c.fetchIncidentAlerts(gctx, id, limit)
			if err != nil {
				return err
			}
			results[i] = IncidentAlertsOutput{IncidentID: id, Alerts: alerts, Total: total}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("unable to retrieve alerts: %w", err)
	}

	return results, nil
}

// ListSimilarIncidents finds similar historical incidents for a given incident
func (c *Client) ListSimilarIncidents(ctx context.Context, incidentID string, limit int) (*ListIncidentsOutput, error) {
	if limit <= 0 {
		limit = defaultQueryLimit
	}

	requestBody := map[string]any{
		"incident_id": incidentID,
		"p":           1,
		"limit":       limit,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/past/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to find similar incidents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []RawIncident `json:"items"`
			Total int           `json:"total"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	if result.Data == nil || len(result.Data.Items) == 0 {
		return &ListIncidentsOutput{
			Incidents: []EnrichedIncident{},
			Total:     0,
		}, nil
	}

	enrichedIncidents, err := c.enrichIncidents(ctx, result.Data.Items)
	if err != nil {
		return nil, fmt.Errorf("unable to load additional incident details: %w", err)
	}

	return &ListIncidentsOutput{
		Incidents: enrichedIncidents,
		Total:     result.Data.Total,
	}, nil
}

// CreateIncidentInput contains parameters for creating an incident
type CreateIncidentInput struct {
	Title       string // Required. Length: 3-200 characters
	Severity    string // Required. Info, Warning, Critical
	ChannelID   int64  // Optional collaboration space ID
	Description string // Optional. Max 6144 characters
	AssignedTo  []int  // Optional person IDs to assign as responders
}

// CreateIncident creates a new incident
func (c *Client) CreateIncident(ctx context.Context, input *CreateIncidentInput) (any, error) {
	requestBody := map[string]any{
		"title":             input.Title,
		"incident_severity": input.Severity,
	}
	if input.ChannelID > 0 {
		requestBody["channel_id"] = input.ChannelID
	}
	if input.Description != "" {
		requestBody["description"] = input.Description
	}
	if len(input.AssignedTo) > 0 {
		requestBody["assigned_to"] = map[string]any{
			"type":       "assign",
			"person_ids": input.AssignedTo,
		}
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/create", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result FlashdutyResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return result.Data, nil
}

// UpdateIncidentInput contains parameters for updating an incident
type UpdateIncidentInput struct {
	IncidentID   string         // Required
	Title        string         // Optional, empty = skip
	Description  string         // Optional
	Severity     string         // Optional
	CustomFields map[string]any // Optional
}

// UpdateIncident updates an incident's fields. Returns the list of updated field names.
func (c *Client) UpdateIncident(ctx context.Context, input *UpdateIncidentInput) ([]string, error) {
	updatedFields := make([]string, 0)

	if input.Title != "" {
		if err := c.updateIncidentField(ctx, input.IncidentID, "/incident/title/reset", "title", input.Title); err != nil {
			return nil, fmt.Errorf("unable to update title: %w", err)
		}
		updatedFields = append(updatedFields, "title")
	}

	if input.Description != "" {
		if err := c.updateIncidentField(ctx, input.IncidentID, "/incident/description/reset", "description", input.Description); err != nil {
			return nil, fmt.Errorf("unable to update description: %w", err)
		}
		updatedFields = append(updatedFields, "description")
	}

	if input.Severity != "" {
		if err := c.updateIncidentField(ctx, input.IncidentID, "/incident/severity/reset", "incident_severity", input.Severity); err != nil {
			return nil, fmt.Errorf("unable to update severity: %w", err)
		}
		updatedFields = append(updatedFields, "severity")
	}

	if len(input.CustomFields) > 0 {
		for fieldName, fieldValue := range input.CustomFields {
			if fieldName == "" {
				return nil, fmt.Errorf("custom field name must not be empty")
			}
			for _, c := range fieldName {
				isValid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
				if !isValid {
					return nil, fmt.Errorf("custom field name '%s' contains invalid characters (only alphanumeric and underscore allowed)", fieldName)
				}
			}
			if err := c.updateCustomField(ctx, input.IncidentID, fieldName, fieldValue); err != nil {
				return nil, fmt.Errorf("unable to update custom field '%s': %w", fieldName, err)
			}
			updatedFields = append(updatedFields, fieldName)
		}
	}

	if len(updatedFields) == 0 {
		return nil, fmt.Errorf("no fields specified to update")
	}

	return updatedFields, nil
}

// AckIncidents acknowledges one or more incidents
func (c *Client) AckIncidents(ctx context.Context, incidentIDs []string) error {
	requestBody := map[string]any{
		"incident_ids": incidentIDs,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/ack", requestBody)
	if err != nil {
		return fmt.Errorf("unable to acknowledge incidents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c.logger, resp)
	}

	var result FlashdutyResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// CloseIncidents closes (resolves) one or more incidents
func (c *Client) CloseIncidents(ctx context.Context, incidentIDs []string) error {
	requestBody := map[string]any{
		"incident_ids": incidentIDs,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/resolve", requestBody)
	if err != nil {
		return fmt.Errorf("unable to close incidents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c.logger, resp)
	}

	var result FlashdutyResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// fetchIncidentsByIDs fetches incidents by their IDs
func (c *Client) fetchIncidentsByIDs(ctx context.Context, incidentIDs []string) ([]RawIncident, error) {
	requestBody := map[string]any{
		"incident_ids": incidentIDs,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/list-by-ids", requestBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []RawIncident `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", result.Error.Code, result.Error.Message)
	}
	if result.Data == nil {
		return nil, nil
	}
	return result.Data.Items, nil
}

// fetchIncidentsByFilters fetches incidents by filters
func (c *Client) fetchIncidentsByFilters(ctx context.Context, progress, severity string, channelID, startTime, endTime int64, title string, limit, page int) ([]RawIncident, error) {
	requestBody := map[string]any{
		"p":          page,
		"limit":      limit,
		"start_time": startTime,
		"end_time":   endTime,
	}

	if progress != "" {
		requestBody["progress"] = progress
	}
	if severity != "" {
		requestBody["incident_severity"] = severity
	}
	if channelID > 0 {
		requestBody["channel_id"] = channelID
	}
	if title != "" {
		requestBody["title"] = title
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/list", requestBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []RawIncident `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", result.Error.Code, result.Error.Message)
	}
	if result.Data == nil {
		return nil, nil
	}
	return result.Data.Items, nil
}

// updateIncidentField is a helper to update a single incident field
func (c *Client) updateIncidentField(ctx context.Context, incidentID, endpoint, fieldName, fieldValue string) error {
	requestBody := map[string]any{
		"incident_id": incidentID,
		fieldName:     fieldValue,
	}

	resp, err := c.makeRequest(ctx, "POST", endpoint, requestBody)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c.logger, resp)
	}

	var result FlashdutyResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return fmt.Errorf("API error: %s - %s", result.Error.Code, result.Error.Message)
	}
	return nil
}

// GetIncidentDetailInput contains parameters for getting incident detail
type GetIncidentDetailInput struct {
	IncidentID string // Required
}

// GetIncidentDetailOutput contains full incident detail
type GetIncidentDetailOutput struct {
	Incident IncidentDetail `json:"incident"`
}

// GetIncidentDetail fetches detailed information for a single incident
func (c *Client) GetIncidentDetail(ctx context.Context, input *GetIncidentDetailInput) (*GetIncidentDetailOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("incident detail input is required")
	}

	requestBody := map[string]any{
		"incident_id": input.IncidentID,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/info", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident detail: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError      `json:"error,omitempty"`
		Data  *IncidentDetail `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	if result.Data == nil {
		return nil, fmt.Errorf("incident not found: %s", input.IncidentID)
	}

	return &GetIncidentDetailOutput{Incident: *result.Data}, nil
}

// GetIncidentFeedInput contains parameters for getting incident feed/timeline
type GetIncidentFeedInput struct {
	IncidentID string // Required
	Limit      int    // Max results (default 20)
	Page       int    // Page number (default 1)
	Asc        bool   // Sort ascending by time (default false)
}

// GetIncidentFeedOutput contains incident feed events
type GetIncidentFeedOutput struct {
	Items       []TimelineEvent `json:"items"`
	HasNextPage bool            `json:"has_next_page"`
}

// GetIncidentFeed fetches the feed/timeline for an incident with enriched person names
func (c *Client) GetIncidentFeed(ctx context.Context, input *GetIncidentFeedInput) (*GetIncidentFeedOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("incident feed input is required")
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
		"incident_id": input.IncidentID,
		"limit":       limit,
		"p":           page,
		"asc":         input.Asc,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/feed", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items       []RawTimelineItem `json:"items"`
			HasNextPage bool              `json:"has_next_page"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	if result.Data == nil || len(result.Data.Items) == 0 {
		return &GetIncidentFeedOutput{
			Items:       []TimelineEvent{},
			HasNextPage: false,
		}, nil
	}

	// Enrich with person names
	personIDs := collectTimelinePersonIDs(result.Data.Items)
	personMap, err := c.fetchPersonInfos(ctx, personIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load person details for feed: %w", err)
	}

	enrichedItems := enrichTimelineItems(result.Data.Items, personMap)

	return &GetIncidentFeedOutput{
		Items:       enrichedItems,
		HasNextPage: result.Data.HasNextPage,
	}, nil
}

// ListPostMortemsInput contains parameters for listing post-mortem reports
type ListPostMortemsInput struct {
	// Deprecated: /incident/post-mortem/list does not support incident_id filtering.
	IncidentID string

	Status                string  // drafting or published
	TeamIDs               []int64 // Optional team filter
	ChannelIDs            []int64 // Optional channel filter
	CreatedAtStartSeconds int64   // Optional creation time lower bound
	CreatedAtEndSeconds   int64   // Optional creation time upper bound
	OrderBy               string  // created_at_seconds or updated_at_seconds
	Asc                   bool    // Sort ascending when true
	Limit                 int     // Max results (default 20)
	Page                  int     // Page number (default 1)
	SearchAfterCtx        string  // Cursor for the next page
}

// ListPostMortemsOutput contains post-mortem reports
type ListPostMortemsOutput struct {
	PostMortems    []PostMortem `json:"post_mortems"`
	Total          int          `json:"total"`
	HasNextPage    bool         `json:"has_next_page"`
	SearchAfterCtx string       `json:"search_after_ctx,omitempty"`
}

// ListPostMortems queries post-mortem reports
func (c *Client) ListPostMortems(ctx context.Context, input *ListPostMortemsInput) (*ListPostMortemsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("post-mortem query input is required")
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
		"limit": limit,
		"p":     page,
	}
	if input.Status != "" {
		requestBody["status"] = input.Status
	}
	if len(input.TeamIDs) > 0 {
		requestBody["team_ids"] = input.TeamIDs
	}
	if len(input.ChannelIDs) > 0 {
		requestBody["channel_ids"] = input.ChannelIDs
	}
	if input.CreatedAtStartSeconds > 0 {
		requestBody["created_at_start_seconds"] = input.CreatedAtStartSeconds
	}
	if input.CreatedAtEndSeconds > 0 {
		requestBody["created_at_end_seconds"] = input.CreatedAtEndSeconds
	}
	if input.OrderBy != "" {
		requestBody["order_by"] = input.OrderBy
	}
	if input.Asc {
		requestBody["asc"] = true
	}
	if input.SearchAfterCtx != "" {
		requestBody["search_after_ctx"] = input.SearchAfterCtx
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/post-mortem/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to list post-mortems: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items          []PostMortem `json:"items"`
			Total          int          `json:"total"`
			HasNextPage    bool         `json:"has_next_page"`
			SearchAfterCtx string       `json:"search_after_ctx,omitempty"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	postMortems := []PostMortem{}
	total := 0
	hasNextPage := false
	searchAfterCtx := ""
	if result.Data != nil {
		postMortems = result.Data.Items
		total = result.Data.Total
		hasNextPage = result.Data.HasNextPage
		searchAfterCtx = result.Data.SearchAfterCtx
	}

	return &ListPostMortemsOutput{
		PostMortems:    postMortems,
		Total:          total,
		HasNextPage:    hasNextPage,
		SearchAfterCtx: searchAfterCtx,
	}, nil
}

// updateCustomField is a helper to update a custom field
func (c *Client) updateCustomField(ctx context.Context, incidentID, fieldName string, fieldValue any) error {
	requestBody := map[string]any{
		"incident_id": incidentID,
		"field_name":  fieldName,
		"field_value": fieldValue,
	}

	resp, err := c.makeRequest(ctx, "POST", "/incident/field/reset", requestBody)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c.logger, resp)
	}

	var result FlashdutyResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return fmt.Errorf("API error: %s - %s", result.Error.Code, result.Error.Message)
	}
	return nil
}
