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

// TestMonitAgentCatalog_RequestShape verifies the wire-level request body for
// /monit/tools/catalog: target_locator + optional target_kind.
func TestMonitAgentCatalog_RequestShape(t *testing.T) {
	t.Parallel()

	gotPath := ""
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"tools": []any{}},
		})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitAgentCatalog(context.Background(), &MonitAgentCatalogInput{
		TargetLocator: "host:web-01",
		TargetKind:    "host",
	}); err != nil {
		t.Fatalf("MonitAgentCatalog: %v", err)
	}

	if gotPath != "/monit/tools/catalog" {
		t.Fatalf("path = %q, want /monit/tools/catalog", gotPath)
	}

	wantBody := map[string]any{
		"target_locator": "host:web-01",
		"target_kind":    "host",
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		gotJSON, _ := json.MarshalIndent(gotBody, "", "  ")
		wantJSON, _ := json.MarshalIndent(wantBody, "", "  ")
		t.Fatalf("request body mismatch\n got:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

// TestMonitAgentCatalog_OmitsTargetKind drops the optional target_kind when
// the agent should infer it.
func TestMonitAgentCatalog_OmitsTargetKind(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"tools": []any{}}})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitAgentCatalog(context.Background(), &MonitAgentCatalogInput{
		TargetLocator: "mysql:prod-master",
	}); err != nil {
		t.Fatalf("MonitAgentCatalog: %v", err)
	}
	if _, ok := gotBody["target_kind"]; ok {
		t.Errorf("target_kind should be omitted when empty, got %v", gotBody["target_kind"])
	}
}

// TestMonitAgentCatalog_Roundtrip ensures tool entries decode through
// MonitAgentTool with the input_schema preserved as RawMessage.
func TestMonitAgentCatalog_Roundtrip(t *testing.T) {
	t.Parallel()

	canned := map[string]any{
		"data": map[string]any{
			"tools": []map[string]any{
				{
					"name":        "ps_top",
					"description": "List top processes by CPU",
					"input_schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"limit": map[string]any{"type": "integer"},
						},
					},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(canned)
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	out, err := client.MonitAgentCatalog(context.Background(), &MonitAgentCatalogInput{
		TargetLocator: "host:web-01",
	})
	if err != nil {
		t.Fatalf("MonitAgentCatalog: %v", err)
	}
	if len(out.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(out.Tools))
	}
	tool := out.Tools[0]
	if tool.Name != "ps_top" || tool.Description != "List top processes by CPU" {
		t.Errorf("tool basics mismatch: %#v", tool)
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("decode input_schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("input_schema mismatch: %v", schema)
	}
}

// TestMonitAgentInvoke_RequestShape verifies the wire-level request body for
// /monit/tools/invoke: target_locator, optional target_kind, tools[]{tool,params}.
func TestMonitAgentInvoke_RequestShape(t *testing.T) {
	t.Parallel()

	gotPath := ""
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"results": []any{}},
		})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitAgentInvoke(context.Background(), &MonitAgentInvokeInput{
		TargetLocator: "host:web-01",
		TargetKind:    "host",
		Tools: []MonitAgentInvokeTool{
			{Tool: "ps_top", Params: json.RawMessage(`{"limit":5}`)},
			{Tool: "disk_usage", Params: json.RawMessage(`{"path":"/"}`)},
		},
	}); err != nil {
		t.Fatalf("MonitAgentInvoke: %v", err)
	}

	if gotPath != "/monit/tools/invoke" {
		t.Fatalf("path = %q, want /monit/tools/invoke", gotPath)
	}

	wantBody := map[string]any{
		"target_locator": "host:web-01",
		"target_kind":    "host",
		"tools": []any{
			map[string]any{
				"tool":   "ps_top",
				"params": map[string]any{"limit": float64(5)},
			},
			map[string]any{
				"tool":   "disk_usage",
				"params": map[string]any{"path": "/"},
			},
		},
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		gotJSON, _ := json.MarshalIndent(gotBody, "", "  ")
		wantJSON, _ := json.MarshalIndent(wantBody, "", "  ")
		t.Fatalf("request body mismatch\n got:\n%s\nwant:\n%s", gotJSON, wantJSON)
	}
}

// TestMonitAgentInvoke_OmitsTargetKindAndAllowsNullParams covers two edges:
// target_kind omitted when empty, and Params left nil (omitted entirely so
// the server applies tool defaults).
func TestMonitAgentInvoke_OmitsTargetKindAndAllowsNullParams(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"results": []any{}}})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	if _, err := client.MonitAgentInvoke(context.Background(), &MonitAgentInvokeInput{
		TargetLocator: "mysql:prod-master",
		Tools: []MonitAgentInvokeTool{
			{Tool: "show_status"},
		},
	}); err != nil {
		t.Fatalf("MonitAgentInvoke: %v", err)
	}
	if _, ok := gotBody["target_kind"]; ok {
		t.Errorf("target_kind should be omitted, got %v", gotBody["target_kind"])
	}
	tools, _ := gotBody["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	tool0, _ := tools[0].(map[string]any)
	if _, ok := tool0["params"]; ok {
		t.Errorf("params should be omitted when nil, got %v", tool0["params"])
	}
}

// TestMonitAgentInvoke_Roundtrip ensures per-tool results decode through
// MonitAgentInvokeResult, preserving per-tool data + error so callers can
// distinguish the three error layers (HTTP, request-level, per-tool).
func TestMonitAgentInvoke_Roundtrip(t *testing.T) {
	t.Parallel()

	canned := map[string]any{
		"data": map[string]any{
			"results": []map[string]any{
				{
					"tool": "ps_top",
					"data": map[string]any{
						"summary": "top processes: chrome (cpu 25%), java (cpu 18%)",
						"data": map[string]any{
							"rows": []map[string]any{
								{"pid": 1, "cpu": 25.0, "cmd": "chrome"},
							},
						},
					},
				},
				{
					"tool":  "ssh_exec",
					"error": "command not allowed",
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(canned)
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	out, err := client.MonitAgentInvoke(context.Background(), &MonitAgentInvokeInput{
		TargetLocator: "host:web-01",
		Tools: []MonitAgentInvokeTool{
			{Tool: "ps_top"},
			{Tool: "ssh_exec"},
		},
	})
	if err != nil {
		t.Fatalf("MonitAgentInvoke: %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out.Results))
	}

	// First tool succeeded — data populated, error empty.
	if out.Results[0].Tool != "ps_top" {
		t.Errorf("results[0].tool = %q, want ps_top", out.Results[0].Tool)
	}
	if out.Results[0].Error != "" {
		t.Errorf("results[0].error should be empty, got %q", out.Results[0].Error)
	}
	if len(out.Results[0].Data) == 0 {
		t.Fatalf("results[0].data should be non-empty")
	}
	var first map[string]any
	if err := json.Unmarshal(out.Results[0].Data, &first); err != nil {
		t.Fatalf("decode results[0].data: %v", err)
	}
	if first["summary"] != "top processes: chrome (cpu 25%), java (cpu 18%)" {
		t.Errorf("results[0] summary mismatch: %v", first["summary"])
	}

	// Second tool failed — error populated, data empty/null.
	if out.Results[1].Tool != "ssh_exec" || out.Results[1].Error != "command not allowed" {
		t.Errorf("results[1] mismatch: %#v", out.Results[1])
	}
}

// TestMonitAgentInvoke_RequestLevelError verifies request-level errors
// (target_unavailable, ambiguous_target_kind, …) surface via the dataEnvelope
// error field and abort before per-tool results are touched.
func TestMonitAgentInvoke_RequestLevelError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "ambiguous_target_kind",
				"message": "specify target_kind explicitly",
			},
		})
	}))
	defer ts.Close()

	client, _ := NewClient("test-key", WithBaseURL(ts.URL))
	_, err := client.MonitAgentInvoke(context.Background(), &MonitAgentInvokeInput{
		TargetLocator: "ambiguous",
		Tools:         []MonitAgentInvokeTool{{Tool: "ps_top"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous_target_kind") {
		t.Errorf("unexpected error: %v", err)
	}
}
