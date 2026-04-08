package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestClientSetUserAgent(t *testing.T) {
	tests := []struct {
		name         string
		initialUA    string // set via WithUserAgent; empty means use default
		dynamicUA    string // set via SetUserAgent before the call
		wantUA       string
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
