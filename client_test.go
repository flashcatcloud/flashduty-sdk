package flashduty

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// capturedHeaders stores the headers received by the test server.
type capturedHeaders struct {
	mu      sync.Mutex
	headers http.Header
}

func (c *capturedHeaders) set(h http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers = h.Clone()
}

func (c *capturedHeaders) get() http.Header {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.headers.Clone()
}

type traceLogEntry struct {
	level string
	msg   string
	kv    []any
}

type capturingLogger struct {
	mu      sync.Mutex
	entries []traceLogEntry
}

func (l *capturingLogger) Debug(msg string, keysAndValues ...any) {
	l.add("debug", msg, keysAndValues...)
}
func (l *capturingLogger) Info(msg string, keysAndValues ...any) {
	l.add("info", msg, keysAndValues...)
}
func (l *capturingLogger) Warn(msg string, keysAndValues ...any) {
	l.add("warn", msg, keysAndValues...)
}
func (l *capturingLogger) Error(msg string, keysAndValues ...any) {
	l.add("error", msg, keysAndValues...)
}

func (l *capturingLogger) add(level, msg string, keysAndValues ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	kvCopy := append([]any(nil), keysAndValues...)
	l.entries = append(l.entries, traceLogEntry{level: level, msg: msg, kv: kvCopy})
}

func (l *capturingLogger) snapshot() []traceLogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]traceLogEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

func findLogEntry(entries []traceLogEntry, msg string) (traceLogEntry, bool) {
	for _, entry := range entries {
		if entry.msg == msg {
			return entry, true
		}
	}
	return traceLogEntry{}, false
}

func traceIDFromKV(kv []any) (string, bool) {
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok || key != "trace_id" {
			continue
		}
		value, ok := kv[i+1].(string)
		if !ok {
			return "", false
		}
		return value, true
	}
	return "", false
}

// newTestServer returns an httptest.Server that captures request headers and
// responds with a valid ListTeams JSON payload.
func newTestServer(cap *capturedHeaders) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.set(r.Header)
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"data": map[string]any{
				"items": []any{},
				"total": 0,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// callListTeams is a helper that invokes ListTeams with an empty name filter
// to trigger a simple POST request through makeRequest.
func callListTeams(t *testing.T, c *Client) {
	t.Helper()
	_, err := c.ListTeams(context.Background(), &ListTeamsInput{Name: "any"})
	if err != nil {
		t.Fatalf("ListTeams returned unexpected error: %v", err)
	}
}

func TestListTeamsByIDsPreservesMembers(t *testing.T) {
	cap := &capturedHeaders{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/team/infos":
			var body struct {
				TeamIDs []int64 `json:"team_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(body.TeamIDs) != 2 || body.TeamIDs[0] != 101 || body.TeamIDs[1] != 202 {
				t.Fatalf("unexpected team_ids payload: %#v", body.TeamIDs)
			}
			cap.set(r.Header)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"items": []any{
						map[string]any{
							"team_id":    101,
							"team_name":  "alpha",
							"person_ids": []int64{1},
						},
					},
				},
			})
		case "/person/infos":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"items": []any{
						map[string]any{
							"person_id":   1,
							"person_name": "Ada",
							"email":       "ada@example.com",
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	out, err := client.ListTeams(context.Background(), &ListTeamsInput{
		TeamIDs: []int64{101, 202},
	})
	if err != nil {
		t.Fatalf("ListTeams returned unexpected error: %v", err)
	}

	if len(out.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(out.Teams))
	}
	team := out.Teams[0]
	if team.TeamID != 101 || team.TeamName != "alpha" {
		t.Fatalf("unexpected team: %#v", team)
	}
	if len(team.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(team.Members))
	}
	member := team.Members[0]
	if member.PersonID != 1 || member.PersonName != "Ada" || member.Email != "ada@example.com" {
		t.Fatalf("unexpected member: %#v", member)
	}
	if out.Total != 1 {
		t.Fatalf("expected total 1, got %d", out.Total)
	}
}

func TestClientWithRequestHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		want    map[string]string // header key -> expected value
	}{
		{
			name: "single custom header",
			headers: http.Header{
				"X-Custom-Trace": []string{"trace-abc-123"},
			},
			want: map[string]string{
				"X-Custom-Trace": "trace-abc-123",
			},
		},
		{
			name: "multiple custom headers",
			headers: http.Header{
				"X-Request-Id":   []string{"req-001"},
				"X-Tenant-Id":    []string{"tenant-42"},
				"X-Custom-Token": []string{"tok-xyz"},
			},
			want: map[string]string{
				"X-Request-Id":   "req-001",
				"X-Tenant-Id":    "tenant-42",
				"X-Custom-Token": "tok-xyz",
			},
		},
		{
			name:    "nil headers (no-op)",
			headers: nil,
			want:    map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cap := &capturedHeaders{}
			ts := newTestServer(cap)
			defer ts.Close()

			client, err := NewClient("test-key",
				WithBaseURL(ts.URL),
				WithRequestHeaders(tc.headers),
			)
			if err != nil {
				t.Fatalf("NewClient error: %v", err)
			}

			callListTeams(t, client)

			got := cap.get()
			for key, wantVal := range tc.want {
				if gotVal := got.Get(key); gotVal != wantVal {
					t.Errorf("header %q = %q; want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestClientWithRequestHook(t *testing.T) {
	tests := []struct {
		name string
		hook func(*http.Request)
		want map[string]string
	}{
		{
			name: "hook injects traceparent",
			hook: func(r *http.Request) {
				r.Header.Set("Traceparent", "00-abcdef1234567890-0123456789abcdef-01")
			},
			want: map[string]string{
				"Traceparent": "00-abcdef1234567890-0123456789abcdef-01",
			},
		},
		{
			name: "hook injects multiple headers",
			hook: func(r *http.Request) {
				r.Header.Set("X-Hook-A", "alpha")
				r.Header.Set("X-Hook-B", "beta")
			},
			want: map[string]string{
				"X-Hook-A": "alpha",
				"X-Hook-B": "beta",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cap := &capturedHeaders{}
			ts := newTestServer(cap)
			defer ts.Close()

			client, err := NewClient("test-key",
				WithBaseURL(ts.URL),
				WithRequestHook(tc.hook),
			)
			if err != nil {
				t.Fatalf("NewClient error: %v", err)
			}

			callListTeams(t, client)

			got := cap.get()
			for key, wantVal := range tc.want {
				if gotVal := got.Get(key); gotVal != wantVal {
					t.Errorf("header %q = %q; want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestClientLogsTraceID(t *testing.T) {
	t.Parallel()

	const traceID = "0123456789abcdef0123456789abcdef"
	const traceparent = "00-" + traceID + "-0123456789abcdef-01"

	cap := &capturedHeaders{}
	logger := &capturingLogger{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.set(r.Header)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{},
				"total": 0,
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient("test-key",
		WithBaseURL(ts.URL),
		WithLogger(logger),
		WithRequestHook(func(r *http.Request) {
			r.Header.Set("traceparent", traceparent)
		}),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	callListTeams(t, client)

	gotHeaders := cap.get()
	if got := gotHeaders.Get("traceparent"); got != traceparent {
		t.Fatalf("traceparent header = %q, want %q", got, traceparent)
	}

	entries := logger.snapshot()
	for _, msg := range []string{"duty request", "duty response"} {
		entry, ok := findLogEntry(entries, msg)
		if !ok {
			t.Fatalf("expected %q log entry, got %#v", msg, entries)
		}
		gotTraceID, ok := traceIDFromKV(entry.kv)
		if !ok {
			t.Fatalf("expected trace_id in %q log entry, got %#v", msg, entry.kv)
		}
		if gotTraceID != traceID {
			t.Fatalf("trace_id in %q log entry = %q, want %q", msg, gotTraceID, traceID)
		}
	}
}

func TestClientLogsTraceIDOnError(t *testing.T) {
	t.Parallel()

	const traceID = "fedcba9876543210fedcba9876543210"
	const traceparent = "00-" + traceID + "-0123456789abcdef-01"

	logger := &capturingLogger{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "InternalError",
				"message": "boom",
			},
		})
	}))
	defer ts.Close()

	client, err := NewClient("test-key",
		WithBaseURL(ts.URL),
		WithLogger(logger),
		WithRequestHook(func(r *http.Request) {
			r.Header.Set("traceparent", traceparent)
		}),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, err = client.ListTeams(context.Background(), &ListTeamsInput{Name: "any"})
	if err == nil {
		t.Fatal("expected error from server, got nil")
	}

	entries := logger.snapshot()
	for _, msg := range []string{"duty request", "duty error"} {
		entry, ok := findLogEntry(entries, msg)
		if !ok {
			t.Fatalf("expected %q log entry, got %#v", msg, entries)
		}
		gotTraceID, ok := traceIDFromKV(entry.kv)
		if !ok {
			t.Fatalf("expected trace_id in %q log entry, got %#v", msg, entry.kv)
		}
		if gotTraceID != traceID {
			t.Fatalf("trace_id in %q log entry = %q, want %q", msg, gotTraceID, traceID)
		}
	}
}

func TestClientSetUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		initialUA string // set via WithUserAgent; empty means use default
		dynamicUA string // set via SetUserAgent before the call
		wantUA    string
	}{
		{
			name:      "override default user agent",
			dynamicUA: "my-custom-agent/2.0",
			wantUA:    "my-custom-agent/2.0",
		},
		{
			name:      "override WithUserAgent option",
			initialUA: "initial-agent/1.0",
			dynamicUA: "updated-agent/3.0",
			wantUA:    "updated-agent/3.0",
		},
		{
			name:      "set to empty string falls back to Go default",
			initialUA: "initial-agent/1.0",
			dynamicUA: "",
			wantUA:    "Go-http-client/1.1", // Go's net/http sets this when no User-Agent header is present
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cap := &capturedHeaders{}
			ts := newTestServer(cap)
			defer ts.Close()

			opts := []Option{WithBaseURL(ts.URL)}
			if tc.initialUA != "" {
				opts = append(opts, WithUserAgent(tc.initialUA))
			}

			client, err := NewClient("test-key", opts...)
			if err != nil {
				t.Fatalf("NewClient error: %v", err)
			}

			client.SetUserAgent(tc.dynamicUA)
			callListTeams(t, client)

			got := cap.get()
			gotUA := got.Get("User-Agent")
			if gotUA != tc.wantUA {
				t.Errorf("User-Agent = %q; want %q", gotUA, tc.wantUA)
			}
		})
	}
}

func TestClientStaticHeadersAndHookBothApplied(t *testing.T) {
	cap := &capturedHeaders{}
	ts := newTestServer(cap)
	defer ts.Close()

	staticHeaders := http.Header{
		"X-Static": []string{"from-static"},
	}

	hook := func(r *http.Request) {
		r.Header.Set("X-Hook", "from-hook")
	}

	client, err := NewClient("test-key",
		WithBaseURL(ts.URL),
		WithRequestHeaders(staticHeaders),
		WithRequestHook(hook),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	callListTeams(t, client)

	got := cap.get()

	if v := got.Get("X-Static"); v != "from-static" {
		t.Errorf("X-Static = %q; want %q", v, "from-static")
	}
	if v := got.Get("X-Hook"); v != "from-hook" {
		t.Errorf("X-Hook = %q; want %q", v, "from-hook")
	}
}

func TestClientHookOverridesStaticHeaders(t *testing.T) {
	cap := &capturedHeaders{}
	ts := newTestServer(cap)
	defer ts.Close()

	staticHeaders := http.Header{
		"X-Overlap": []string{"static-value"},
	}

	hook := func(r *http.Request) {
		// The hook runs after static headers, so this should win.
		r.Header.Set("X-Overlap", "hook-value")
	}

	client, err := NewClient("test-key",
		WithBaseURL(ts.URL),
		WithRequestHeaders(staticHeaders),
		WithRequestHook(hook),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	callListTeams(t, client)

	got := cap.get()
	if v := got.Get("X-Overlap"); v != "hook-value" {
		t.Errorf("X-Overlap = %q; want %q (hook should override static header)", v, "hook-value")
	}
}

func TestSanitizeBodyRedactsSensitiveKeys(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		mustNotAppear []string
		mustAppear    []string
	}{
		{
			name:          "api_key",
			input:         `{"api_key":"secret-123","source_page_id":"p1"}`,
			mustNotAppear: []string{"secret-123"},
			mustAppear:    []string{"[REDACTED]", "p1"},
		},
		{
			name:          "password",
			input:         `{"password":"hunter2","user":"ada"}`,
			mustNotAppear: []string{"hunter2"},
			mustAppear:    []string{"[REDACTED]", "ada"},
		},
		{
			name:          "token",
			input:         `{"token":"abcd","other":"x"}`,
			mustNotAppear: []string{"abcd"},
			mustAppear:    []string{"[REDACTED]", "x"},
		},
		{
			name:          "secret",
			input:         `{"secret":"sshh","x":1}`,
			mustNotAppear: []string{"sshh"},
			mustAppear:    []string{"[REDACTED]"},
		},
		{
			name:       "no_sensitive_keys_preserved",
			input:      `{"page_id":42,"title":"hi"}`,
			mustAppear: []string{"page_id", "42", "hi"},
		},
		{
			name:       "empty_input_passthrough",
			input:      "",
			mustAppear: nil,
		},
		{
			name:       "non_json_passthrough",
			input:      "raw=not-json",
			mustAppear: []string{"raw=not-json"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeBody(tc.input)
			for _, s := range tc.mustNotAppear {
				if strings.Contains(got, s) {
					t.Errorf("sanitizeBody(%q) = %q; must not contain %q", tc.input, got, s)
				}
			}
			for _, s := range tc.mustAppear {
				if !strings.Contains(got, s) {
					t.Errorf("sanitizeBody(%q) = %q; want to contain %q", tc.input, got, s)
				}
			}
		})
	}
}

func TestSanitizeBodyRedactsNestedCaseInsensitiveAliases(t *testing.T) {
	input := `{"outer":{"ApiKey":"secret-123","nested":[{"ACCESS_TOKEN":"token-abc"},{"safe":"ok"}],"credentials":{"clientSecret":"super-secret"}},"Authorization":"Bearer top-secret","page_id":"p1"}`

	got := sanitizeBody(input)

	for _, secret := range []string{"secret-123", "token-abc", "super-secret", "Bearer top-secret"} {
		if strings.Contains(got, secret) {
			t.Errorf("sanitizeBody(%q) = %q; must not contain %q", input, got, secret)
		}
	}

	for _, want := range []string{"[REDACTED]", `"safe":"ok"`, `"page_id":"p1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("sanitizeBody(%q) = %q; want to contain %q", input, got, want)
		}
	}
}

// env and headers maps carry user-chosen keys (e.g. OPENAI_API_KEY,
// X-Prom-Token) whose names the static allow-list cannot anticipate. Every
// value under these parents must be redacted unconditionally, while sibling
// fields stay visible so the log still indicates payload shape.
func TestSanitizeBodyRedactsAllChildrenOfEnvAndHeaders(t *testing.T) {
	input := `{"env":{"OPENAI_API_KEY":"sk-secret","FOO":"bar"},"headers":{"X-Custom":"tok"},"server_name":"prom-prod"}`

	got := sanitizeBody(input)

	for _, secret := range []string{"sk-secret", "bar", "tok"} {
		if strings.Contains(got, secret) {
			t.Errorf("sanitizeBody(%q) = %q; must not contain %q", input, got, secret)
		}
	}

	for _, want := range []string{
		`"OPENAI_API_KEY":"[REDACTED]"`,
		`"FOO":"[REDACTED]"`,
		`"X-Custom":"[REDACTED]"`,
		`"server_name":"prom-prod"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sanitizeBody(%q) = %q; want to contain %q", input, got, want)
		}
	}
}

func TestMakeRequestLogsRedactedBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"job_id": "j1"}})
	}))
	t.Cleanup(ts.Close)

	logger := &capturingLogger{}
	client, err := NewClient("app-key", WithBaseURL(ts.URL), WithLogger(logger))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "atlassian-secret",
		SourcePageID: "page_123",
	})
	if err != nil {
		t.Fatalf("StartStatusPageMigration: %v", err)
	}

	entries := logger.snapshot()
	req, ok := findLogEntry(entries, "duty request")
	if !ok {
		t.Fatalf("expected a 'duty request' log entry; got %d entries", len(entries))
	}

	body := logKVString(req.kv, "body")
	if body == "" {
		t.Fatalf("body field missing from log entry")
	}
	if strings.Contains(body, "atlassian-secret") {
		t.Errorf("request log leaked api_key: body = %q", body)
	}
	if !strings.Contains(body, "[REDACTED]") {
		t.Errorf("request log missing redaction marker: body = %q", body)
	}
}

func TestParseResponseLogsRedactedBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"job_id": "j1",
				"echo": map[string]any{
					"ApiKey": "response-secret",
					"nested": []any{
						map[string]any{"AUTHORIZATION": "Bearer response-token"},
					},
				},
			},
		})
	}))
	t.Cleanup(ts.Close)

	logger := &capturingLogger{}
	client, err := NewClient("app-key", WithBaseURL(ts.URL), WithLogger(logger))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "request-secret",
		SourcePageID: "page_123",
	})
	if err != nil {
		t.Fatalf("StartStatusPageMigration: %v", err)
	}

	entries := logger.snapshot()
	resp, ok := findLogEntry(entries, "duty response")
	if !ok {
		t.Fatalf("expected a 'duty response' log entry; got %d entries", len(entries))
	}

	body := logKVString(resp.kv, "body")
	if body == "" {
		t.Fatalf("body field missing from log entry")
	}
	for _, secret := range []string{"response-secret", "Bearer response-token"} {
		if strings.Contains(body, secret) {
			t.Errorf("response log leaked secret %q: body = %q", secret, body)
		}
	}
	if !strings.Contains(body, "[REDACTED]") {
		t.Errorf("response log missing redaction marker: body = %q", body)
	}
}

func TestHandleAPIErrorLogsRedactedBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid credentials",
				"details": map[string]any{
					"clientSecret": "response-secret",
					"nested": []any{
						map[string]any{"ACCESS_TOKEN": "response-token"},
					},
				},
			},
		})
	}))
	t.Cleanup(ts.Close)

	logger := &capturingLogger{}
	client, err := NewClient("app-key", WithBaseURL(ts.URL), WithLogger(logger))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.StartStatusPageMigration(context.Background(), &StartStatusPageMigrationInput{
		SourceAPIKey: "request-secret",
		SourcePageID: "page_123",
	})
	if err == nil {
		t.Fatal("expected StartStatusPageMigration to fail, got nil")
	}

	entries := logger.snapshot()
	resp, ok := findLogEntry(entries, "duty error")
	if !ok {
		t.Fatalf("expected a 'duty error' log entry; got %d entries", len(entries))
	}

	body := logKVString(resp.kv, "body")
	if body == "" {
		t.Fatalf("body field missing from log entry")
	}
	for _, secret := range []string{"response-secret", "response-token"} {
		if strings.Contains(body, secret) {
			t.Errorf("error log leaked secret %q: body = %q", secret, body)
		}
	}
	if !strings.Contains(body, "[REDACTED]") {
		t.Errorf("error log missing redaction marker: body = %q", body)
	}
}

func TestParseResponseReturnsSanitizedErrorBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     http.Header{"Flashcat-Request-Id": []string{"req-1"}},
		Body: io.NopCloser(strings.NewReader(
			`{"error":{"message":"upstream failed","details":{"ApiKey":"response-secret","nested":[{"ACCESS_TOKEN":"response-token"}]}}}`,
		)),
		Request: &http.Request{Header: make(http.Header)},
	}

	err := parseResponse(&capturingLogger{}, resp, nil)
	if err == nil {
		t.Fatal("expected parseResponse to fail, got nil")
	}
	if !strings.HasPrefix(err.Error(), "API server error (HTTP 502, request_id: req-1): ") {
		t.Fatalf("parseResponse returned unexpected error format: %v", err)
	}
	if strings.Contains(err.Error(), "response-secret") || strings.Contains(err.Error(), "response-token") {
		t.Fatalf("parseResponse leaked secret in error: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("parseResponse error missing redaction marker: %v", err)
	}
}

func TestHandleAPIErrorReturnsSanitizedErrorBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{"Flashcat-Request-Id": []string{"req-2"}},
		Body: io.NopCloser(strings.NewReader(
			`{"error":{"message":"invalid credentials","details":{"clientSecret":"response-secret","nested":[{"AUTHORIZATION":"Bearer response-token"}]}}}`,
		)),
		Request: &http.Request{Header: make(http.Header)},
	}

	err := handleAPIError(&capturingLogger{}, resp)
	if err == nil {
		t.Fatal("expected handleAPIError to fail, got nil")
	}
	if !strings.HasPrefix(err.Error(), "API client error (HTTP 403, request_id: req-2): ") {
		t.Fatalf("handleAPIError returned unexpected error format: %v", err)
	}
	if strings.Contains(err.Error(), "response-secret") || strings.Contains(err.Error(), "response-token") {
		t.Fatalf("handleAPIError leaked secret in error: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("handleAPIError error missing redaction marker: %v", err)
	}
}

func TestReturnedAPIErrorsPreserveNonJSONBody(t *testing.T) {
	tests := []struct {
		name string
		call func(*http.Response) error
		want string
	}{
		{
			name: "parseResponse",
			call: func(resp *http.Response) error {
				return parseResponse(&capturingLogger{}, resp, nil)
			},
			want: "API client error (HTTP 400, request_id: req-plain-parse): upstream returned plaintext secret=response-secret",
		},
		{
			name: "handleAPIError",
			call: func(resp *http.Response) error {
				return handleAPIError(&capturingLogger{}, resp)
			},
			want: "API client error (HTTP 400, request_id: req-plain-handle): upstream returned plaintext secret=response-secret",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("upstream returned plaintext secret=response-secret")),
				Request:    &http.Request{Header: make(http.Header)},
			}
			if tc.name == "parseResponse" {
				resp.Header.Set("Flashcat-Request-Id", "req-plain-parse")
			} else {
				resp.Header.Set("Flashcat-Request-Id", "req-plain-handle")
			}

			err := tc.call(resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); got != tc.want {
				t.Fatalf("error = %q; want %q", got, tc.want)
			}
		})
	}
}

func logKVString(kv []any, key string) string {
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok && k == key {
			if v, ok := kv[i+1].(string); ok {
				return v
			}
		}
	}
	return ""
}
