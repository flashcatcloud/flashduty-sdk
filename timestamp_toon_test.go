package flashduty

import (
	"strings"
	"testing"
	"time"
)

// TOON is a primary LLM output format for the CLI/MCP. toon-go renders scalars
// via fmt.Stringer for named types, so Timestamp/TimestampMilli must produce
// RFC3339 there too — not error out or emit a raw integer.

func TestTimestamp_TOON_RendersRFC3339(t *testing.T) {
	withLocal(t, "UTC")
	type row struct {
		StartTime Timestamp      `json:"start_time" toon:"start_time"`
		Created   TimestampMilli `json:"created" toon:"created"`
		AckTime   Timestamp      `json:"ack_time,omitempty" toon:"ack_time,omitempty"`
	}
	r := row{
		StartTime: Timestamp(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).Unix()),
		Created:   TimestampMilli(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).UnixMilli()),
		// AckTime zero + omitempty -> dropped
	}
	b, err := Marshal(r, OutputFormatTOON)
	if err != nil {
		t.Fatalf("toon marshal errored (Timestamp not TOON-renderable): %v", err)
	}
	s := string(b)
	if strings.Count(s, "2026-05-28T08:00:00") < 2 {
		t.Errorf("toon did not render both timestamps as RFC3339:\n%s", s)
	}
	if strings.Contains(s, "ack_time") {
		t.Errorf("omitempty zero Timestamp should be dropped in TOON:\n%s", s)
	}
}
