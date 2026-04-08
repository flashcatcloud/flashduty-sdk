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
		rawIncidents, err = c.fetchIncidentsByFilters(ctx, input.Progress, input.Severity, input.ChannelID, input.StartTime, input.EndTime, input.Title, limit)
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
func (c *Client) fetchIncidentsByFilters(ctx context.Context, progress, severity string, channelID, startTime, endTime int64, title string, limit int) ([]RawIncident, error) {
	requestBody := map[string]any{
		"p":          1,
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
