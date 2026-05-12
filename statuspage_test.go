package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestStartStatusPageMigration(t *testing.T) {
	var gotMethod, gotPath, gotAppKey, gotContentType string
	var gotBody map[string]any

	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAppKey = r.URL.Query().Get("app_key")
		gotContentType = r.Header.Get("Content-Type")
		gotBody = decodeJSONBody(t, r)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"job_id": "job-1"},
		})
	})

	out, err := client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "atlassian-key",
		SourcePageID: "page_123",
	})
	if err != nil {
		t.Fatalf("StartStatusPageMigration() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/status-page/migrate-structure" {
		t.Errorf("path = %s, want /status-page/migrate-structure", gotPath)
	}
	if gotAppKey != "test-key" {
		t.Errorf("app_key = %s, want test-key", gotAppKey)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", gotContentType)
	}
	if gotBody["api_key"] != "atlassian-key" {
		t.Errorf("api_key = %v, want atlassian-key", gotBody["api_key"])
	}
	if gotBody["source_page_id"] != "page_123" {
		t.Errorf("source_page_id = %v, want page_123", gotBody["source_page_id"])
	}
	if _, ok := gotBody["url_name"]; ok {
		t.Errorf("url_name should be omitted when empty, got %v", gotBody["url_name"])
	}
	if out.JobID != "job-1" {
		t.Errorf("JobID = %s, want job-1", out.JobID)
	}
}

func TestStartStatusPageMigrationSendsURLName(t *testing.T) {
	var gotBody map[string]any

	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody = decodeJSONBody(t, r)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"job_id": "job-url"},
		})
	})

	out, err := client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "atlassian-key",
		SourcePageID: "page_123",
		URLName:      "desired-page",
	})
	if err != nil {
		t.Fatalf("StartStatusPageMigration() error = %v", err)
	}

	if gotBody["url_name"] != "desired-page" {
		t.Errorf("url_name = %v, want desired-page", gotBody["url_name"])
	}
	if out.JobID != "job-url" {
		t.Errorf("JobID = %s, want job-url", out.JobID)
	}
}

func TestStartStatusPageEmailSubscriberMigration(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeJSONBody(t, r)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"job_id": "job-2"},
		})
	})

	out, err := client.StartStatusPageEmailSubscriberMigration(context.Background(), &StartStatusPageEmailSubscriberMigrationInput{
		SourceAPIKey: "atlassian-key",
		SourcePageID: "page_123",
		TargetPageID: 1024,
	})
	if err != nil {
		t.Fatalf("StartStatusPageEmailSubscriberMigration() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/status-page/migrate-email-subscribers" {
		t.Errorf("path = %s, want /status-page/migrate-email-subscribers", gotPath)
	}
	if gotBody["api_key"] != "atlassian-key" {
		t.Errorf("api_key = %v, want atlassian-key", gotBody["api_key"])
	}
	if gotBody["source_page_id"] != "page_123" {
		t.Errorf("source_page_id = %v, want page_123", gotBody["source_page_id"])
	}
	// JSON decoding of map[string]any produces float64 for numbers.
	if got, ok := gotBody["target_page_id"].(float64); !ok || int64(got) != 1024 {
		t.Errorf("target_page_id = %#v, want 1024", gotBody["target_page_id"])
	}
	if out.JobID != "job-2" {
		t.Errorf("JobID = %s, want job-2", out.JobID)
	}
}

func TestGetStatusPageMigrationStatus(t *testing.T) {
	var gotMethod, gotPath, gotJobID string

	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotJobID = r.URL.Query().Get("job_id")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"job_id":         "job-3",
				"source_page_id": "src-1",
				"target_page_id": 2048,
				"phase":          "history",
				"status":         "running",
				"progress": map[string]any{
					"total_steps":           5,
					"completed_steps":       3,
					"components_imported":   10,
					"sections_imported":     2,
					"incidents_imported":    4,
					"maintenances_imported": 1,
					"subscribers_imported":  50,
					"subscribers_skipped":   3,
					"templates_imported":    2,
					"warnings":              []string{"missing field X"},
				},
				"created_at": 1713225600,
				"updated_at": 1713225700,
			},
		})
	})

	out, err := client.GetStatusPageMigrationStatus(context.Background(), "job-3")
	if err != nil {
		t.Fatalf("GetStatusPageMigrationStatus() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/status-page/migration/status" {
		t.Errorf("path = %s, want /status-page/migration/status", gotPath)
	}
	if gotJobID != "job-3" {
		t.Errorf("job_id query = %s, want job-3", gotJobID)
	}

	if out.JobID != "job-3" {
		t.Errorf("JobID = %s, want job-3", out.JobID)
	}
	if out.SourcePageID != "src-1" {
		t.Errorf("SourcePageID = %s, want src-1", out.SourcePageID)
	}
	if out.TargetPageID != 2048 {
		t.Errorf("TargetPageID = %d, want 2048", out.TargetPageID)
	}
	if out.Phase != "history" {
		t.Errorf("Phase = %s, want history", out.Phase)
	}
	if out.Status != "running" {
		t.Errorf("Status = %s, want running", out.Status)
	}
	if out.Progress.CompletedSteps != 3 || out.Progress.TotalSteps != 5 {
		t.Errorf("Progress steps = %d/%d, want 3/5", out.Progress.CompletedSteps, out.Progress.TotalSteps)
	}
	if out.Progress.SubscribersImported != 50 {
		t.Errorf("Progress.SubscribersImported = %d, want 50", out.Progress.SubscribersImported)
	}
	if len(out.Progress.Warnings) != 1 || out.Progress.Warnings[0] != "missing field X" {
		t.Errorf("Progress.Warnings = %v, want [missing field X]", out.Progress.Warnings)
	}
	if out.CreatedAt != 1713225600 || out.UpdatedAt != 1713225700 {
		t.Errorf("timestamps = (%d, %d)", out.CreatedAt, out.UpdatedAt)
	}
}

func TestGetStatusPageMigrationStatusRejectsEmptyJobID(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be hit for empty jobID; got %s %s", r.Method, r.URL.Path)
	})

	if _, err := client.GetStatusPageMigrationStatus(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty jobID, got nil")
	}
}

func TestCancelStatusPageMigration(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody = decodeJSONBody(t, r)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	})

	if err := client.CancelStatusPageMigration(context.Background(), "job-4"); err != nil {
		t.Fatalf("CancelStatusPageMigration() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/status-page/migration/cancel" {
		t.Errorf("path = %s, want /status-page/migration/cancel", gotPath)
	}
	if gotBody["job_id"] != "job-4" {
		t.Errorf("job_id = %v, want job-4", gotBody["job_id"])
	}
}

func TestCancelStatusPageMigrationRejectsEmptyJobID(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be hit for empty jobID; got %s %s", r.Method, r.URL.Path)
	})

	if err := client.CancelStatusPageMigration(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty jobID, got nil")
	}
}

func TestStatusPageMigrationReturnsDutyError(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "invalid_api_key",
				"message": "source provider API key is invalid",
			},
		})
	})

	_, err := client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "bad",
		SourcePageID: "page_123",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	dutyErr, ok := err.(*DutyError)
	if !ok {
		t.Fatalf("error type = %T, want *DutyError (err: %v)", err, err)
	}
	if dutyErr.Code != "invalid_api_key" {
		t.Errorf("Code = %s, want invalid_api_key", dutyErr.Code)
	}
	if dutyErr.Message != "source provider API key is invalid" {
		t.Errorf("Message = %s, want source provider API key is invalid", dutyErr.Message)
	}
}

func TestStatusPageMigrationWrapsHTTPError(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal","message":"boom"}}`))
	})

	_, err := client.GetStatusPageMigrationStatus(context.Background(), "job-5")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// handleAPIError wraps the HTTP status into the error message.
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v; want it to mention HTTP 500", err)
	}
}

func TestStartStatusPageMigrationErrorsOnMissingData(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	_, err := client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "k",
		SourcePageID: "p",
	})
	if err == nil || !strings.Contains(err.Error(), "missing data") {
		t.Fatalf("expected missing-data error, got %v", err)
	}
}

func TestGetStatusPageMigrationStatusErrorsOnMissingData(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	_, err := client.GetStatusPageMigrationStatus(context.Background(), "job-6")
	if err == nil || !strings.Contains(err.Error(), "missing data") {
		t.Fatalf("expected missing-data error, got %v", err)
	}
}

func TestCancelStatusPageMigrationReturnsDutyError(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "not_found", "message": "job not found"},
		})
	})

	err := client.CancelStatusPageMigration(context.Background(), "missing-job")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if _, ok := err.(*DutyError); !ok {
		t.Errorf("error type = %T, want *DutyError", err)
	}
}

func TestStartStatusPageMigrationRejectsNilInput(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be hit for nil input; got %s %s", r.Method, r.URL.Path)
	})

	if _, err := client.StartStatusPageMigration(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil input, got nil")
	}
	if _, err := client.StartStatusPageEmailSubscriberMigration(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil input, got nil")
	}
}

func TestCancelStatusPageMigrationWrapsHTTPError(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"forbidden","message":"nope"}}`))
	})

	err := client.CancelStatusPageMigration(context.Background(), "job-7")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %v; want it to mention HTTP 403", err)
	}
}
