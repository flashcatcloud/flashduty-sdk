package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestChannelEnumValues(t *testing.T) {
	t.Parallel()
	channels := ChannelEnumValues()
	if len(channels) != 13 {
		t.Fatalf("expected 13 channels, got %d", len(channels))
	}

	want := map[string]bool{
		"dingtalk": true, "slack": true, "telegram": true,
		"email": true, "sms": true, "zoom": true,
	}
	found := make(map[string]bool)
	for _, ch := range channels {
		found[ch] = true
	}
	if !slices.IsSorted(channels) {
		t.Fatal("expected channel enum values to be sorted")
	}
	for k := range want {
		if !found[k] {
			t.Errorf("missing expected channel %q", k)
		}
	}
}

func TestTemplateVariables(t *testing.T) {
	t.Parallel()
	vars := TemplateVariables()
	if len(vars) != 40 {
		t.Fatalf("expected 40 template variables, got %d", len(vars))
	}

	// Verify it returns a copy
	vars[0].Name = "MUTATED"
	original := TemplateVariables()
	if original[0].Name == "MUTATED" {
		t.Fatal("TemplateVariables() should return a copy, not the original slice")
	}
}

func TestTemplateCustomFunctions(t *testing.T) {
	t.Parallel()
	fns := TemplateCustomFunctions()
	if len(fns) != 19 {
		t.Fatalf("expected 19 custom functions, got %d", len(fns))
	}

	// Verify it returns a copy
	fns[0].Name = "MUTATED"
	original := TemplateCustomFunctions()
	if original[0].Name == "MUTATED" {
		t.Fatal("TemplateCustomFunctions() should return a copy, not the original slice")
	}
}

func TestTemplateSprigFunctions(t *testing.T) {
	t.Parallel()
	fns := TemplateSprigFunctions()
	if len(fns) != 19 {
		t.Fatalf("expected 19 sprig functions, got %d", len(fns))
	}

	// Verify it returns a copy
	fns[0].Name = "MUTATED"
	original := TemplateSprigFunctions()
	if original[0].Name == "MUTATED" {
		t.Fatal("TemplateSprigFunctions() should return a copy, not the original slice")
	}
}

func TestGetPresetTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		channel    string
		serverResp map[string]any
		wantErr    string
		wantCode   string
	}{
		{
			name:    "invalid channel",
			channel: "unknown",
			wantErr: "unknown channel: unknown",
		},
		{
			name:    "valid channel returns template",
			channel: "slack",
			serverResp: map[string]any{
				"data": map[string]any{
					"slack": "Hello {{.Title}}",
				},
			},
			wantCode: "Hello {{.Title}}",
		},
		{
			name:    "empty template code",
			channel: "slack",
			serverResp: map[string]any{
				"data": map[string]any{
					"slack": "",
				},
			},
			wantErr: "no preset template found for channel: slack",
		},
		{
			name:    "API error",
			channel: "slack",
			serverResp: map[string]any{
				"error": map[string]any{
					"code":    "Forbidden",
					"message": "access denied",
				},
			},
			wantErr: "Forbidden: access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ts *httptest.Server
			if tt.serverResp != nil {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(tt.serverResp)
				}))
				defer ts.Close()
			}

			opts := []Option{}
			if ts != nil {
				opts = append(opts, WithBaseURL(ts.URL))
			}
			client, err := NewClient("test-key", opts...)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			out, err := client.GetPresetTemplate(context.Background(), &GetPresetTemplateInput{
				Channel: tt.channel,
			})

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.TemplateCode != tt.wantCode {
				t.Errorf("template code = %q, want %q", out.TemplateCode, tt.wantCode)
			}
			if out.Channel != tt.channel {
				t.Errorf("channel = %q, want %q", out.Channel, tt.channel)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        *ValidateTemplateInput
		serverResp   map[string]any
		wantErr      string
		wantSuccess  bool
		wantErrors   int
		wantWarnings int
	}{
		{
			name:    "invalid channel",
			input:   &ValidateTemplateInput{Channel: "unknown", TemplateCode: "test"},
			wantErr: "unknown channel: unknown",
		},
		{
			name:  "successful validation",
			input: &ValidateTemplateInput{Channel: "email", TemplateCode: "Hello {{.Title}}"},
			serverResp: map[string]any{
				"data": map[string]any{
					"success": true,
					"content": "Hello Test Incident",
					"message": "",
				},
			},
			wantSuccess: true,
		},
		{
			name:  "template parse error",
			input: &ValidateTemplateInput{Channel: "email", TemplateCode: "{{.Bad"},
			serverResp: map[string]any{
				"data": map[string]any{
					"success": false,
					"content": "",
					"message": "template parse error",
				},
			},
			wantSuccess: false,
			wantErrors:  1,
		},
		{
			name:  "size limit exceeded - telegram",
			input: &ValidateTemplateInput{Channel: "telegram", TemplateCode: "big"},
			serverResp: map[string]any{
				"data": map[string]any{
					"success": true,
					"content": strings.Repeat("x", 5000),
					"message": "",
				},
			},
			wantSuccess: false,
			wantErrors:  1,
		},
		{
			name:  "size warning at 80% threshold",
			input: &ValidateTemplateInput{Channel: "dingtalk", TemplateCode: "medium"},
			serverResp: map[string]any{
				"data": map[string]any{
					"success": true,
					"content": strings.Repeat("x", 3500), // 87.5% of 4000
					"message": "",
				},
			},
			wantSuccess:  true,
			wantWarnings: 1,
		},
		{
			name:  "no limit channel - no warning",
			input: &ValidateTemplateInput{Channel: "email", TemplateCode: "big"},
			serverResp: map[string]any{
				"data": map[string]any{
					"success": true,
					"content": strings.Repeat("x", 100000),
					"message": "",
				},
			},
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ts *httptest.Server
			if tt.serverResp != nil {
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(tt.serverResp)
				}))
				defer ts.Close()
			}

			opts := []Option{}
			if ts != nil {
				opts = append(opts, WithBaseURL(ts.URL))
			}
			client, err := NewClient("test-key", opts...)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			out, err := client.ValidateTemplate(context.Background(), tt.input)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Success != tt.wantSuccess {
				t.Errorf("success = %v, want %v (errors: %v, warnings: %v)", out.Success, tt.wantSuccess, out.Errors, out.Warnings)
			}
			if len(out.Errors) != tt.wantErrors {
				t.Errorf("errors count = %d, want %d: %v", len(out.Errors), tt.wantErrors, out.Errors)
			}
			if len(out.Warnings) != tt.wantWarnings {
				t.Errorf("warnings count = %d, want %d: %v", len(out.Warnings), tt.wantWarnings, out.Warnings)
			}
		})
	}
}
