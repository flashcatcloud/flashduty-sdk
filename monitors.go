package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

// QueryMonitorRuleStatusInput contains parameters for querying monitor rule status
type QueryMonitorRuleStatusInput struct {
	// Deprecated: /monit/rule/counter/status does not currently accept request filters.
	TeamID int64
}

// MonitorRuleFolderStatus contains monitor rule counts for a single folder family.
type MonitorRuleFolderStatus struct {
	FolderID           int64  `json:"folder_id"`
	FolderName         string `json:"folder_name,omitempty"`
	RuleTotal          int64  `json:"rule_total"`
	TriggeredRuleCount int64  `json:"triggered_rule_count"`
}

// QueryMonitorRuleStatusOutput contains monitor rule status summaries.
type QueryMonitorRuleStatusOutput struct {
	Statuses []MonitorRuleFolderStatus `json:"statuses"`
}

// QueryMonitorRuleStatus queries monitor rule status counts grouped by folder family.
func (c *Client) QueryMonitorRuleStatus(ctx context.Context, input *QueryMonitorRuleStatusInput) (*QueryMonitorRuleStatusOutput, error) {
	if input == nil {
		input = &QueryMonitorRuleStatusInput{}
	}

	requestBody := map[string]any{}

	resp, err := c.makeRequest(ctx, "POST", "/monit/rule/counter/status", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitor rule status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError                `json:"error,omitempty"`
		Data  []MonitorRuleFolderStatus `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	statuses := []MonitorRuleFolderStatus{}
	if result.Data != nil {
		statuses = result.Data
	}

	return &QueryMonitorRuleStatusOutput{Statuses: statuses}, nil
}
