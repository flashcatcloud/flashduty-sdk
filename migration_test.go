package flashduty

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// These tests lock in the int64 -> Timestamp/TimestampMilli migration: they fail
// if a migrated instant field is reverted to a raw integer, or if a trap field
// (duration/offset) is wrongly converted to a timestamp.

func TestMigration_IncidentDetail_SecondsRenderRFC3339(t *testing.T) {
	withLocal(t, "UTC")
	d := IncidentDetail{
		IncidentID: "abc",
		StartTime:  Timestamp(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).Unix()),
		// AckTime / CloseTime left zero (omitempty) -> must be dropped, not "1970"
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"start_time":"2026-05-28T08:00:00Z"`) {
		t.Errorf("start_time not RFC3339: %s", s)
	}
	if strings.Contains(s, "ack_time") || strings.Contains(s, "close_time") {
		t.Errorf("zero omitempty time fields should be dropped, not rendered: %s", s)
	}
}

func TestMigration_TimelineEvent_MillisRenderRFC3339(t *testing.T) {
	withLocal(t, "UTC")
	e := TimelineEvent{
		Type:      "ack",
		Timestamp: TimestampMilli(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).UnixMilli()),
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"timestamp":"2026-05-28T08:00:00Z"`) {
		t.Errorf("ms timestamp not RFC3339: %s", b)
	}
}

func TestMigration_ScheduleLayer_TrapsStayNumeric_InstantsRender(t *testing.T) {
	withLocal(t, "UTC")
	layer := ScheduleLayer{
		RotationDuration: 86400, // duration -> MUST stay numeric
		HandoffTime:      3600,  // within-cycle offset -> numeric
		RestrictEnd:      3600,  // cyclic-window offset -> numeric
		EnableTime:       Timestamp(time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC).Unix()),
		LayerStart:       Timestamp(time.Date(2026, 5, 28, 9, 0, 0, 0, time.UTC).Unix()),
	}
	b, err := json.Marshal(layer)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, num := range []string{`"rotation_duration":86400`, `"handoff_time":3600`, `"restrict_end":3600`} {
		if !strings.Contains(s, num) {
			t.Errorf("trap field not left numeric (want %s): %s", num, s)
		}
	}
	if !strings.Contains(s, `"enable_time":"2026-05-28T08:00:00Z"`) {
		t.Errorf("enable_time not RFC3339: %s", s)
	}
	if !strings.Contains(s, `"layer_start":"2026-05-28T09:00:00Z"`) {
		t.Errorf("layer_start not RFC3339: %s", s)
	}
}

func TestMigration_CreateIncidentOutput_Decode(t *testing.T) {
	var out CreateIncidentOutput
	body := `{"incident_id":"69db2ef1a0fe7db6448b14f1","title":"API test incident for docs"}`
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.IncidentID != "69db2ef1a0fe7db6448b14f1" || out.Title != "API test incident for docs" {
		t.Errorf("got %+v", out)
	}
}

func TestMigration_CreateStatusIncidentOutput_Decode(t *testing.T) {
	var out CreateStatusIncidentOutput
	body := `{"change_id":6294539747131,"change_name":"API Test Incident"}`
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ChangeID != 6294539747131 || out.ChangeName != "API Test Incident" {
		t.Errorf("got %+v", out)
	}
}
