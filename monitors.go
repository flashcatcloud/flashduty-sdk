package flashduty

import (
	"context"
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
func (c *Client) QueryMonitorRuleStatus(ctx context.Context, _ *QueryMonitorRuleStatusInput) (*QueryMonitorRuleStatusOutput, error) {
	requestBody := map[string]any{}

	statuses, err := postData[[]MonitorRuleFolderStatus](c, ctx, "/monit/rule/counter/status", requestBody, "failed to query monitor rule status")
	if err != nil {
		return nil, err
	}

	out := []MonitorRuleFolderStatus{}
	if statuses != nil {
		out = *statuses
	}

	return &QueryMonitorRuleStatusOutput{Statuses: out}, nil
}
