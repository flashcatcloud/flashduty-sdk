package flashduty

import (
	"context"
	"fmt"
)

// SearchAuditLogsInput contains parameters for searching audit logs
type SearchAuditLogsInput struct {
	StartTime int64 // Required: Unix seconds
	EndTime   int64 // Required: Unix seconds
	Limit     int   // Max results (default 20)

	RequestID      string   // Optional: filter by request ID
	SearchAfterCtx string   // Optional: opaque cursor for the next page
	Operations     []string // Optional: filter by operation names
	PersonID       int64    // Optional: filter by operator person ID
	IsDangerous    *bool    // Optional: filter high-risk operations
	IsWrite        *bool    // Optional: filter read/write operations

	// Deprecated: use Operations.
	Operation string
	// Deprecated: use PersonID.
	OperatorID int64
	// Deprecated: /audit/search uses SearchAfterCtx for pagination.
	Page int
}

// AuditLogParam represents a single path parameter captured in an audit log entry.
type AuditLogParam struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// AuditLogRecord represents a single audit log entry returned by /audit/search.
type AuditLogRecord struct {
	CreatedAt     TimestampMilli  `json:"created_at"`
	AccountID     int64           `json:"account_id"`
	MemberID      int64           `json:"member_id"`
	MemberName    string          `json:"member_name"`
	RequestID     string          `json:"request_id"`
	IP            string          `json:"ip"`
	Operation     string          `json:"operation"`
	OperationName string          `json:"operation_name"`
	Body          string          `json:"body"`
	Params        []AuditLogParam `json:"params"`
	IsDangerous   bool            `json:"is_dangerous"`
	IsWrite       bool            `json:"is_write"`
}

// SearchAuditLogsOutput contains audit log entries plus the next-page cursor.
type SearchAuditLogsOutput struct {
	AuditLogs      []AuditLogRecord `json:"audit_logs"`
	Total          int64            `json:"total"`
	SearchAfterCtx string           `json:"search_after_ctx"`
}

// SearchAuditLogs searches the audit log
func (c *Client) SearchAuditLogs(ctx context.Context, input *SearchAuditLogsInput) (*SearchAuditLogsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("search audit logs input is required")
	}
	if input.StartTime <= 0 || input.EndTime <= 0 {
		return nil, fmt.Errorf("start_time and end_time are required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}

	requestBody := map[string]any{
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
		"limit":      limit,
	}
	if input.RequestID != "" {
		requestBody["request_id"] = input.RequestID
	}
	if input.SearchAfterCtx != "" {
		requestBody["search_after_ctx"] = input.SearchAfterCtx
	}
	operations := input.Operations
	if len(operations) == 0 && input.Operation != "" {
		operations = []string{input.Operation}
	}
	if len(operations) > 0 {
		requestBody["operations"] = operations
	}
	personID := input.PersonID
	if personID <= 0 && input.OperatorID > 0 {
		personID = input.OperatorID
	}
	if personID > 0 {
		requestBody["person_id"] = personID
	}
	if input.IsDangerous != nil {
		requestBody["is_dangerous"] = *input.IsDangerous
	}
	if input.IsWrite != nil {
		requestBody["is_write"] = *input.IsWrite
	}

	result, err := postData[struct {
		Docs           []AuditLogRecord `json:"docs"`
		Total          int64            `json:"total"`
		SearchAfterCtx string           `json:"search_after_ctx"`
	}](c, ctx, "/audit/search", requestBody, "failed to search audit logs")
	if err != nil {
		return nil, err
	}

	logs := []AuditLogRecord{}
	total := int64(0)
	searchAfterCtx := ""
	if result != nil {
		logs = result.Docs
		total = result.Total
		searchAfterCtx = result.SearchAfterCtx
	}

	return &SearchAuditLogsOutput{
		AuditLogs:      logs,
		Total:          total,
		SearchAfterCtx: searchAfterCtx,
	}, nil
}
