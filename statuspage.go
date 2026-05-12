package flashduty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ListStatusPages queries status pages, optionally filtering by page IDs
func (c *Client) ListStatusPages(ctx context.Context, pageIDs []int64) ([]StatusPage, error) {
	resp, err := c.makeRequest(ctx, "GET", "/status-page/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list status pages: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []struct {
				PageID      int64  `json:"page_id"`
				PageName    string `json:"name"`
				URLName     string `json:"url_name,omitempty"`
				Description string `json:"description,omitempty"`
				Components  []struct {
					ComponentID string `json:"component_id"`
					Name        string `json:"name"`
				} `json:"components,omitempty"`
			} `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	if result.Data == nil || len(result.Data.Items) == 0 {
		return []StatusPage{}, nil
	}

	// Build page ID filter set
	pageIDSet := make(map[int64]struct{})
	for _, id := range pageIDs {
		pageIDSet[id] = struct{}{}
	}

	pages := make([]StatusPage, 0)
	for _, item := range result.Data.Items {
		if len(pageIDs) > 0 {
			if _, ok := pageIDSet[item.PageID]; !ok {
				continue
			}
		}

		page := StatusPage{
			PageID:      item.PageID,
			PageName:    item.PageName,
			Slug:        item.URLName,
			Description: item.Description,
		}

		worstStatus := "operational"
		if len(item.Components) > 0 {
			page.Components = make([]StatusComponent, 0, len(item.Components))
			for _, comp := range item.Components {
				page.Components = append(page.Components, StatusComponent{
					ComponentID:   comp.ComponentID,
					ComponentName: comp.Name,
					Status:        "operational",
				})
			}
		}
		page.OverallStatus = worstStatus

		pages = append(pages, page)
	}

	return pages, nil
}

// ListStatusChangesInput contains parameters for listing status page changes
type ListStatusChangesInput struct {
	PageID     int64  // Required
	ChangeType string // Required: "incident" or "maintenance"
}

// ListStatusChangesOutput contains the result of listing status changes
type ListStatusChangesOutput struct {
	Changes []StatusChange `json:"changes"`
	Total   int            `json:"total"`
}

// ListStatusChanges lists active incidents or maintenances on a status page
func (c *Client) ListStatusChanges(ctx context.Context, input *ListStatusChangesInput) (*ListStatusChangesOutput, error) {
	if input.ChangeType != "incident" && input.ChangeType != "maintenance" {
		return nil, fmt.Errorf("type must be 'incident' or 'maintenance'")
	}

	params := url.Values{}
	params.Set("page_id", strconv.FormatInt(input.PageID, 10))
	params.Set("type", input.ChangeType)
	resp, err := c.makeRequest(ctx, "GET", "/status-page/change/active/list?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list status changes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []StatusChange `json:"items"`
			Total int            `json:"total"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	changes := []StatusChange{}
	total := 0
	if result.Data != nil {
		changes = result.Data.Items
		total = result.Data.Total
	}

	return &ListStatusChangesOutput{
		Changes: changes,
		Total:   total,
	}, nil
}

// CreateStatusIncidentInput contains parameters for creating a status page incident
type CreateStatusIncidentInput struct {
	PageID             int64  // Required
	Title              string // Required. Max 255 characters
	Message            string // Optional. Initial update message
	Status             string // Optional. Default: "investigating"
	AffectedComponents string // Optional. Format: "id1:degraded,id2:partial_outage"
	NotifySubscribers  bool   // Whether to notify page subscribers
}

// CreateStatusIncident creates an incident on a status page
func (c *Client) CreateStatusIncident(ctx context.Context, input *CreateStatusIncidentInput) (any, error) {
	status := input.Status
	if status == "" {
		status = "investigating"
	}

	update := map[string]any{
		"at_seconds": time.Now().Unix(),
		"status":     status,
	}
	if input.Message != "" {
		update["description"] = input.Message
	}

	// Parse component changes if provided
	if input.AffectedComponents != "" {
		var componentChanges []map[string]string
		parts := parseCommaSeparatedStrings(input.AffectedComponents)
		for _, part := range parts {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				componentChanges = append(componentChanges, map[string]string{
					"component_id": strings.TrimSpace(kv[0]),
					"status":       strings.TrimSpace(kv[1]),
				})
			} else if len(kv) == 1 && kv[0] != "" {
				componentChanges = append(componentChanges, map[string]string{
					"component_id": strings.TrimSpace(kv[0]),
					"status":       "partial_outage",
				})
			}
		}
		if len(componentChanges) > 0 {
			update["component_changes"] = componentChanges
		}
	}

	description := input.Message
	if description == "" {
		description = input.Title
	}

	requestBody := map[string]any{
		"page_id":            input.PageID,
		"title":              input.Title,
		"type":               "incident",
		"status":             status,
		"description":        description,
		"updates":            []map[string]any{update},
		"notify_subscribers": input.NotifySubscribers,
	}

	resp, err := c.makeRequest(ctx, "POST", "/status-page/change/create", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create status incident: %w", err)
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

// CreateChangeTimelineInput contains parameters for adding a timeline entry
type CreateChangeTimelineInput struct {
	PageID           int64  // Required
	ChangeID         int64  // Required
	Message          string // Required
	AtSeconds        int64  // Optional. Defaults to current time
	Status           string // Optional
	ComponentChanges string // Optional. JSON array of component status changes
}

// CreateChangeTimeline adds a timeline update to a status page incident or maintenance
func (c *Client) CreateChangeTimeline(ctx context.Context, input *CreateChangeTimelineInput) error {
	requestBody := map[string]any{
		"page_id":     input.PageID,
		"change_id":   input.ChangeID,
		"description": input.Message,
	}
	if input.AtSeconds > 0 {
		requestBody["at_seconds"] = input.AtSeconds
	}
	if input.Status != "" {
		requestBody["status"] = input.Status
	}
	if input.ComponentChanges != "" {
		var changes []map[string]string
		if err := json.Unmarshal([]byte(input.ComponentChanges), &changes); err == nil {
			requestBody["component_changes"] = changes
		}
	}

	resp, err := c.makeRequest(ctx, "POST", "/status-page/change/timeline/create", requestBody)
	if err != nil {
		return fmt.Errorf("failed to create timeline: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result FlashdutyResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// StartStatusPageMigrationInput contains parameters for starting a status page
// structure and history migration from an external provider.
type StartStatusPageMigrationInput struct {
	SourceAPIKey string // Required. API key for the source provider (e.g. Atlassian Statuspage)
	SourcePageID string // Required. Page identifier in the source provider
	URLName      string // Optional. URL name to use when creating a new Flashduty public status page
}

// StartStatusPageEmailSubscriberMigrationInput contains parameters for starting
// an email subscriber migration from an external provider into an existing
// Flashduty status page.
type StartStatusPageEmailSubscriberMigrationInput struct {
	SourceAPIKey string // Required. API key for the source provider
	SourcePageID string // Required. Page identifier in the source provider
	TargetPageID int64  // Required. Flashduty status page ID to import into
}

// StartStatusPageMigrationOutput contains the result of starting an async
// status page migration. Both structure and email subscriber migrations return
// a job ID that can be polled with GetStatusPageMigrationStatus.
type StartStatusPageMigrationOutput struct {
	JobID string `json:"job_id"`
}

// StatusPageMigrationProgress describes incremental counters reported by a
// migration job. Fields are populated best-effort; zero values indicate
// either the counter does not apply to the current phase or no items of that
// kind have been processed yet.
type StatusPageMigrationProgress struct {
	TotalSteps           int      `json:"total_steps"`
	CompletedSteps       int      `json:"completed_steps"`
	ComponentsImported   int      `json:"components_imported"`
	SectionsImported     int      `json:"sections_imported"`
	IncidentsImported    int      `json:"incidents_imported"`
	MaintenancesImported int      `json:"maintenances_imported"`
	SubscribersImported  int      `json:"subscribers_imported"`
	SubscribersSkipped   int      `json:"subscribers_skipped"`
	TemplatesImported    int      `json:"templates_imported"`
	Warnings             []string `json:"warnings,omitempty"`
}

// StatusPageMigrationJob describes the state of a status page migration job.
type StatusPageMigrationJob struct {
	JobID        string                      `json:"job_id"`
	SourcePageID string                      `json:"source_page_id"`
	TargetPageID int64                       `json:"target_page_id"`
	Phase        string                      `json:"phase"`
	Status       string                      `json:"status"`
	Progress     StatusPageMigrationProgress `json:"progress"`
	Error        string                      `json:"error,omitempty"`
	CreatedAt    int64                       `json:"created_at"`
	UpdatedAt    int64                       `json:"updated_at"`
}

// StartStatusPageMigration starts an asynchronous migration of status page
// structure and history from an external provider into Flashduty. The returned
// job ID can be polled with GetStatusPageMigrationStatus and cancelled with
// CancelStatusPageMigration.
func (c *Client) StartStatusPageMigration(ctx context.Context, input *StartStatusPageMigrationInput) (*StartStatusPageMigrationOutput, error) {
	if input == nil {
		return nil, errors.New("input is required")
	}
	body := map[string]any{
		"api_key":        input.SourceAPIKey,
		"source_page_id": input.SourcePageID,
	}
	if input.URLName != "" {
		body["url_name"] = input.URLName
	}
	return c.startStatusPageMigration(ctx, "/status-page/migrate-structure", body)
}

// StartStatusPageEmailSubscriberMigration starts an asynchronous migration of
// email subscribers from an external provider into an existing Flashduty
// status page. The returned job ID can be polled with
// GetStatusPageMigrationStatus and cancelled with CancelStatusPageMigration.
func (c *Client) StartStatusPageEmailSubscriberMigration(ctx context.Context, input *StartStatusPageEmailSubscriberMigrationInput) (*StartStatusPageMigrationOutput, error) {
	if input == nil {
		return nil, errors.New("input is required")
	}
	return c.startStatusPageMigration(ctx, "/status-page/migrate-email-subscribers", map[string]any{
		"api_key":        input.SourceAPIKey,
		"source_page_id": input.SourcePageID,
		"target_page_id": input.TargetPageID,
	})
}

func (c *Client) startStatusPageMigration(ctx context.Context, path string, body map[string]any) (*StartStatusPageMigrationOutput, error) {
	resp, err := c.makeRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to start status page migration: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError                      `json:"error,omitempty"`
		Data  *StartStatusPageMigrationOutput `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, errors.New("status page migration response missing data")
	}

	return result.Data, nil
}

// GetStatusPageMigrationStatus fetches the current state of a status page
// migration job identified by jobID.
func (c *Client) GetStatusPageMigrationStatus(ctx context.Context, jobID string) (*StatusPageMigrationJob, error) {
	if jobID == "" {
		return nil, errors.New("jobID is required")
	}

	params := url.Values{}
	params.Set("job_id", jobID)
	resp, err := c.makeRequest(ctx, "GET", "/status-page/migration/status?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get status page migration status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError              `json:"error,omitempty"`
		Data  *StatusPageMigrationJob `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, fmt.Errorf("status page migration status response missing data")
	}

	return result.Data, nil
}

// CancelStatusPageMigration requests cancellation of an in-flight status page
// migration job identified by jobID.
func (c *Client) CancelStatusPageMigration(ctx context.Context, jobID string) error {
	if jobID == "" {
		return errors.New("jobID is required")
	}

	resp, err := c.makeRequest(ctx, "POST", "/status-page/migration/cancel", map[string]any{
		"job_id": jobID,
	})
	if err != nil {
		return fmt.Errorf("failed to cancel status page migration: %w", err)
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
