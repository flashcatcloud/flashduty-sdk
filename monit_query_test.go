package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestMonitQueryDiagnose_RequestShape verifies the wire-level request body
// matches the documented /monit/query/diagnose contract: ds_type, ds_name,
// time_range:{start,end}, input:{query}, operation, optional limits.
func TestMonitQueryDiagnose_RequestShape(t *testing.T) {
	t.Parallel()

	input := &MonitQueryDiagnoseInput{
		DsType:         "prometheus",
		DsName:         "prom-prod",
		TimeStart:      1700000000,
		TimeEnd:        1700000900,
		Operation:      "metric_trends",
		Input:          MonitQueryDiagnoseQuery{Query: `rate(http_requests_total[5m])`},
		MaxLogsScanned: 20000,
		MaxPatterns:    30,
		TimeoutSeconds: 28,
	}

	gotPath := ""
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"operation": "metric_trends",
				"results":   []any{},
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.MonitQueryDiagnose(context.Background(), input); err != nil {
		t.Fatalf("MonitQueryDiagnose: %v", err)
	}

	if gotPath != "/monit/query/diagnose" {
		t.Fatalf("path = %q, want /monit/query/diagnose", gotPath)
	}

	wantBody := map[string]any{
		"ds_type": "prometheus",
		"ds_name": "prom-prod",
		"time_range": map[string]any{
			"start": float64(1700000000),
			"end":   float64(1700000900),
		},
		"input": map[string]any{
			"query": "rate(http_requests_total[5m])",
		},
		"operation":        "metric_trends",
		"max_logs_scanned": float64(20000),
		"max_patterns":     float64(30),
		"timeout_seconds":  float64(28),
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		gotJSON, _ := json.MarshalIndent(gotBody, "", "  ")
		wantJSON, _ := json.MarshalIndent(wantBody, "", "  ")
		t.Fatalf("request body mismatch\n got:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

// TestMonitQueryDiagnose_OmitsZeroOptionals verifies optional caps + operation
// are omitted when unset, leaving the server defaults in effect.
func TestMonitQueryDiagnose_OmitsZeroOptionals(t *testing.T) {
	t.Parallel()

	input := &MonitQueryDiagnoseInput{
		DsType:    "victorialogs",
		DsName:    "vl-prod",
		TimeStart: 1700000000,
		TimeEnd:   1700000900,
		Input:     MonitQueryDiagnoseQuery{Query: "_msg:ERROR"},
	}

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"operation": "log_patterns", "results": []any{}}})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitQueryDiagnose(context.Background(), input); err != nil {
		t.Fatalf("MonitQueryDiagnose: %v", err)
	}

	if _, ok := gotBody["operation"]; ok {
		t.Errorf("operation should be omitted when zero, got %v", gotBody["operation"])
	}
	if _, ok := gotBody["max_logs_scanned"]; ok {
		t.Errorf("max_logs_scanned should be omitted when zero")
	}
	if _, ok := gotBody["max_patterns"]; ok {
		t.Errorf("max_patterns should be omitted when zero")
	}
	if _, ok := gotBody["timeout_seconds"]; ok {
		t.Errorf("timeout_seconds should be omitted when zero")
	}
	tr, ok := gotBody["time_range"].(map[string]any)
	if !ok {
		t.Fatalf("time_range must be sent even with operation omitted, got %v", gotBody["time_range"])
	}
	if tr["start"] != float64(1700000000) || tr["end"] != float64(1700000900) {
		t.Errorf("time_range payload = %v, want start=1700000000 end=1700000900", tr)
	}
}

// TestMonitQueryDiagnose_Roundtrip confirms a canned envelope decodes through
// MonitQueryDiagnoseOutput with Results preserved as RawMessage.
func TestMonitQueryDiagnose_Roundtrip(t *testing.T) {
	t.Parallel()

	canned := map[string]any{
		"data": map[string]any{
			"operation": "log_patterns",
			"results": []map[string]any{
				{"signature": "abc123", "sample_count": 42, "first_seen": 1700000010},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(canned)
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	out, err := client.MonitQueryDiagnose(context.Background(), &MonitQueryDiagnoseInput{
		DsType:    "loki",
		DsName:    "loki-prod",
		TimeStart: 1700000000,
		TimeEnd:   1700000900,
		Input:     MonitQueryDiagnoseQuery{Query: `{app="api"} |= "error"`},
	})
	if err != nil {
		t.Fatalf("MonitQueryDiagnose: %v", err)
	}
	if out.Operation != "log_patterns" {
		t.Errorf("operation = %q, want log_patterns", out.Operation)
	}

	var results []map[string]any
	if err := json.Unmarshal(out.Results, &results); err != nil {
		t.Fatalf("decode results: %v (raw=%s)", err, string(out.Results))
	}
	if len(results) != 1 || results[0]["signature"] != "abc123" {
		t.Errorf("results payload mismatch: %#v", results)
	}
}

// TestMonitQueryDiagnose_ServerError surfaces request-level errors via the
// dataEnvelope wrapper.
func TestMonitQueryDiagnose_ServerError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "BadRequest",
				"message": "time range exceeds 6h",
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	_, err := client.MonitQueryDiagnose(context.Background(), &MonitQueryDiagnoseInput{
		DsType:    "prometheus",
		DsName:    "prom",
		TimeStart: 1,
		TimeEnd:   100000,
		Input:     MonitQueryDiagnoseQuery{Query: "up"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "BadRequest") || !strings.Contains(err.Error(), "time range exceeds 6h") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestMonitQueryRows_RequestShape verifies request marshalling for
// /monit/query/rows: ds_type, ds_name, expr, args (string→string).
func TestMonitQueryRows_RequestShape(t *testing.T) {
	t.Parallel()

	input := &MonitQueryRowsInput{
		DsType: "mysql",
		DsName: "mysql-prod",
		Expr:   "SELECT 1",
		Args:   map[string]string{"host": "db-1", "schema": "public"},
	}

	gotPath := ""
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitQueryRows(context.Background(), input); err != nil {
		t.Fatalf("MonitQueryRows: %v", err)
	}

	if gotPath != "/monit/query/rows" {
		t.Fatalf("path = %q, want /monit/query/rows", gotPath)
	}

	wantBody := map[string]any{
		"ds_type": "mysql",
		"ds_name": "mysql-prod",
		"expr":    "SELECT 1",
		"args": map[string]any{
			"host":   "db-1",
			"schema": "public",
		},
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		gotJSON, _ := json.MarshalIndent(gotBody, "", "  ")
		wantJSON, _ := json.MarshalIndent(wantBody, "", "  ")
		t.Fatalf("request body mismatch\n got:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

// TestMonitQueryRows_OmitsEmptyArgs leaves out the args key when unset.
func TestMonitQueryRows_OmitsEmptyArgs(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitQueryRows(context.Background(), &MonitQueryRowsInput{
		DsType: "prometheus",
		DsName: "prom",
		Expr:   "up",
	}); err != nil {
		t.Fatalf("MonitQueryRows: %v", err)
	}
	if _, ok := gotBody["args"]; ok {
		t.Errorf("args should be omitted when empty, got %v", gotBody["args"])
	}
}

// TestMonitQueryRows_Roundtrip ensures the response payload is captured as
// RawMessage so callers can shape rows per datasource.
func TestMonitQueryRows_Roundtrip(t *testing.T) {
	t.Parallel()

	canned := map[string]any{
		"data": []map[string]any{
			{
				"fields": map[string]any{"col_1": "name", "col_2": "value"},
				"values": map[string]any{"name": "a", "value": 1},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(canned)
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	out, err := client.MonitQueryRows(context.Background(), &MonitQueryRowsInput{
		DsType: "mysql",
		DsName: "mysql-prod",
		Expr:   "SELECT 1",
	})
	if err != nil {
		t.Fatalf("MonitQueryRows: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(out.Data, &rows); err != nil {
		t.Fatalf("decode rows: %v (raw=%s)", err, string(out.Data))
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	fields, _ := rows[0]["fields"].(map[string]any)
	if fields["col_1"] != "name" {
		t.Errorf("fields mismatch: %#v", rows[0])
	}
}
