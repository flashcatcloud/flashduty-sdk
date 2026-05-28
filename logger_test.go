package flashduty

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// logEntry records a single logger call.
type logEntry struct {
	Level         string
	Msg           string
	KeysAndValues []any
}

// recordingLogger is a thread-safe mock that records every call.
type recordingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

func (r *recordingLogger) record(level, msg string, kv []any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, logEntry{Level: level, Msg: msg, KeysAndValues: kv})
}

func (r *recordingLogger) Debug(msg string, kv ...any) { r.record("debug", msg, kv) }
func (r *recordingLogger) Info(msg string, kv ...any)  { r.record("info", msg, kv) }
func (r *recordingLogger) Warn(msg string, kv ...any)  { r.record("warn", msg, kv) }
func (r *recordingLogger) Error(msg string, kv ...any) { r.record("error", msg, kv) }

func (r *recordingLogger) Entries() []logEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]logEntry, len(r.entries))
	copy(cp, r.entries)
	return cp
}

// ---------- Test 1: default slogLogger does not panic ----------

func TestSlogLogger_NoPanic(t *testing.T) {
	t.Parallel()
	l := &slogLogger{}

	tests := []struct {
		name string
		fn   func(string, ...any)
	}{
		{"Debug", l.Debug},
		{"Info", l.Info},
		{"Warn", l.Warn},
		{"Error", l.Error},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Should not panic with no extra args.
			tc.fn("simple message")
			// Should not panic with key-value pairs.
			tc.fn("with kv", "key", "value", "count", 42)
		})
	}
}

// ---------- Test 2: recording mock receives expected calls ----------

func TestRecordingLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		call      func(Logger)
		wantLevel string
		wantMsg   string
		wantKVLen int
	}{
		{
			name:      "Debug with kv",
			call:      func(l Logger) { l.Debug("d msg", "k1", "v1") },
			wantLevel: "debug",
			wantMsg:   "d msg",
			wantKVLen: 2,
		},
		{
			name:      "Info no kv",
			call:      func(l Logger) { l.Info("i msg") },
			wantLevel: "info",
			wantMsg:   "i msg",
			wantKVLen: 0,
		},
		{
			name:      "Warn with multiple kv",
			call:      func(l Logger) { l.Warn("w msg", "a", 1, "b", 2) },
			wantLevel: "warn",
			wantMsg:   "w msg",
			wantKVLen: 4,
		},
		{
			name:      "Error with kv",
			call:      func(l Logger) { l.Error("e msg", "err", "boom") },
			wantLevel: "error",
			wantMsg:   "e msg",
			wantKVLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := &recordingLogger{}
			tc.call(rec)
			entries := rec.Entries()
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(entries))
			}
			e := entries[0]
			if e.Level != tc.wantLevel {
				t.Errorf("level: got %q, want %q", e.Level, tc.wantLevel)
			}
			if e.Msg != tc.wantMsg {
				t.Errorf("msg: got %q, want %q", e.Msg, tc.wantMsg)
			}
			if len(e.KeysAndValues) != tc.wantKVLen {
				t.Errorf("kv len: got %d, want %d", len(e.KeysAndValues), tc.wantKVLen)
			}
		})
	}
}

// ---------- Test 3: WithLogger(nil) preserves default ----------

func TestWithLogger_NilKeepsDefault(t *testing.T) {
	t.Parallel()

	c, err := NewClient("test-key", WithLogger(nil))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, ok := c.logger.(*slogLogger); !ok {
		t.Errorf("expected *slogLogger, got %T", c.logger)
	}
}

// ---------- Test 4: client with custom logger uses it during requests ----------

func TestClient_CustomLoggerUsedInRequest(t *testing.T) {
	t.Parallel()

	// httptest server returns a valid JSON response for /team/list.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"items":[],"total":0}}`)
	}))
	defer ts.Close()

	rec := &recordingLogger{}

	c, err := NewClient("test-key",
		WithBaseURL(ts.URL),
		WithLogger(rec),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.ListTeams(context.Background(), &ListTeamsInput{Name: "test"})
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}

	entries := rec.Entries()
	if len(entries) == 0 {
		t.Fatal("expected logger to be called at least once, got 0 entries")
	}

	// makeRequest logs "duty request" at Info, parseResponse logs "duty response" at Info.
	var foundRequest, foundResponse bool
	for _, e := range entries {
		switch e.Msg {
		case "duty request":
			foundRequest = true
			if e.Level != "info" {
				t.Errorf("duty request: expected level info, got %s", e.Level)
			}
		case "duty response":
			foundResponse = true
			if e.Level != "info" {
				t.Errorf("duty response: expected level info, got %s", e.Level)
			}
		}
	}
	if !foundRequest {
		t.Error("missing 'duty request' log entry")
	}
	if !foundResponse {
		t.Error("missing 'duty response' log entry")
	}
}
