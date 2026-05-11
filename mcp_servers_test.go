package flashduty

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateMCPServer_HappyPath(t *testing.T) {
	t.Parallel()

	var captured CreateMCPServerInput
	var capturedMethod string
	var capturedPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"server_id": "mcp_abc123",
				"status":    "enabled",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	out, err := client.CreateMCPServer(context.Background(), &CreateMCPServerInput{
		ServerName:  "prom-prod",
		Description: "Prometheus prod MCP",
		Transport:   "sse",
		URL:         "https://prom.example/mcp",
		Headers:     map[string]string{"Authorization": "Bearer secret"},
		TeamID:      0,
	})
	if err != nil {
		t.Fatalf("CreateMCPServer error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("HTTP method = %q, want POST", capturedMethod)
	}
	if capturedPath != "/safari/mcp/server/create" {
		t.Errorf("path = %q, want /safari/mcp/server/create", capturedPath)
	}
	if captured.ServerName != "prom-prod" {
		t.Errorf("server_name = %q, want prom-prod", captured.ServerName)
	}
	if captured.Transport != "sse" {
		t.Errorf("transport = %q, want sse", captured.Transport)
	}
	if captured.TeamID != 0 {
		t.Errorf("team_id = %d, want 0", captured.TeamID)
	}

	if out.ServerID != "mcp_abc123" {
		t.Errorf("server_id = %q, want mcp_abc123", out.ServerID)
	}
	if out.Status != "enabled" {
		t.Errorf("status = %q, want enabled", out.Status)
	}
}

func TestCreateMCPServer_APIError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "InvalidArgument",
				"message": "transport must be one of stdio, sse, streamable-http",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.CreateMCPServer(context.Background(), &CreateMCPServerInput{
		ServerName: "bad",
		Transport:  "carrier-pigeon",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Assert SDK unwraps the response envelope rather than just wrapping the
	// raw body — a generic fmt.Errorf would also pass a substring check.
	var de *DutyError
	if !errors.As(err, &de) {
		t.Fatalf("want *DutyError, got %T: %v", err, err)
	}
	if de.Code != "InvalidArgument" {
		t.Fatalf("want code InvalidArgument, got %q", de.Code)
	}
}

func TestCreateMCPServer_HTTPError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"Internal","message":"boom"}}`))
	}))
	defer ts.Close()

	client, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.CreateMCPServer(context.Background(), &CreateMCPServerInput{
		ServerName: "x",
		Transport:  "sse",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
