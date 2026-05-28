package flashduty

import (
	"context"
	"encoding/json"
	"fmt"
)

// MonitQueryDiagnoseQuery is the inner `input` payload of /monit/query/diagnose.
// For log datasources it is a filter-only LogsQL/LogQL query; for metric
// datasources it is a matrix PromQL expression.
type MonitQueryDiagnoseQuery struct {
	Query string `json:"query"`
}

// MonitQueryDiagnoseInput is the request payload for /monit/query/diagnose.
//
// TimeStart and TimeEnd are unix seconds; they marshal into a nested
// time_range:{start,end} object on the wire. The server defaults the time
// range to the last 15m if both are zero and rejects spans > 6h.
//
// MaxLogsScanned, MaxPatterns and TimeoutSeconds are optional caps; the
// server applies defaults (10000 / 20 / 25) when omitted and refuses values
// above 50000 / 50 / 30. Operation is also optional — the agent infers it
// from DsType (`log_patterns` for log datasources, `metric_trends` for
// metric datasources).
type MonitQueryDiagnoseInput struct {
	DsType         string
	DsName         string
	TimeStart      int64
	TimeEnd        int64
	Operation      string
	Input          MonitQueryDiagnoseQuery
	MaxLogsScanned int
	MaxPatterns    int
	TimeoutSeconds int
}

// MonitQueryDiagnoseOutput is the decoded response from /monit/query/diagnose.
// Results varies by Operation (log_patterns vs metric_trends), so it is left
// as RawMessage for callers to decode based on Operation.
type MonitQueryDiagnoseOutput struct {
	Operation string          `json:"operation"`
	Results   json.RawMessage `json:"results"`
}

// MonitQueryDiagnose runs a built-in diagnostic operation (log pattern
// extraction or metric trend analysis) against a configured datasource.
func (c *Client) MonitQueryDiagnose(ctx context.Context, input *MonitQueryDiagnoseInput) (*MonitQueryDiagnoseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("monit query diagnose: input is required")
	}

	requestBody := map[string]any{
		"ds_type": input.DsType,
		"ds_name": input.DsName,
		"time_range": map[string]any{
			"start": input.TimeStart,
			"end":   input.TimeEnd,
		},
		"input": map[string]any{
			"query": input.Input.Query,
		},
	}
	if input.Operation != "" {
		requestBody["operation"] = input.Operation
	}
	if input.MaxLogsScanned > 0 {
		requestBody["max_logs_scanned"] = input.MaxLogsScanned
	}
	if input.MaxPatterns > 0 {
		requestBody["max_patterns"] = input.MaxPatterns
	}
	if input.TimeoutSeconds > 0 {
		requestBody["timeout_seconds"] = input.TimeoutSeconds
	}

	return postData[MonitQueryDiagnoseOutput](c, ctx, "/monit/query/diagnose", requestBody, "failed to run monit query diagnose")
}

// MonitQueryRowsInput is the request payload for /monit/query/rows.
//
// Args carries template parameters for parameterized queries; the server
// contract requires every arg value to be a string (no implicit coercion).
type MonitQueryRowsInput struct {
	DsType string
	DsName string
	Expr   string
	Args   map[string]string
}

// MonitQueryRowsOutput is the decoded response from /monit/query/rows.
// Data shape varies per datasource (PromQL instant vector, MySQL rows, …)
// so it is left as RawMessage for callers to shape.
type MonitQueryRowsOutput struct {
	Data json.RawMessage `json:"-"`
}

// UnmarshalJSON captures the entire response data field as a RawMessage so
// callers can decode per-datasource. The server returns data as a JSON array
// of {fields,values} objects, but the row schema is datasource-specific.
func (o *MonitQueryRowsOutput) UnmarshalJSON(b []byte) error {
	o.Data = append(o.Data[:0], b...)
	return nil
}

// MonitQueryRows executes a raw expression against a datasource and returns
// the rows. The caller is responsible for crafting datasource-appropriate
// expressions (PromQL, LogsQL, LogQL, SQL).
func (c *Client) MonitQueryRows(ctx context.Context, input *MonitQueryRowsInput) (*MonitQueryRowsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("monit query rows: input is required")
	}

	requestBody := map[string]any{
		"ds_type": input.DsType,
		"ds_name": input.DsName,
		"expr":    input.Expr,
	}
	if len(input.Args) > 0 {
		requestBody["args"] = input.Args
	}

	return postData[MonitQueryRowsOutput](c, ctx, "/monit/query/rows", requestBody, "failed to run monit query rows")
}
