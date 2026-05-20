package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetDataDecodesEnvelope(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/common/test" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"name": "ok",
			},
		})
	})

	out, err := getData[struct {
		Name string `json:"name"`
	}](client, context.Background(), "/common/test", "failed to get test data")
	if err != nil {
		t.Fatalf("getData returned error: %v", err)
	}
	if out.Name != "ok" {
		t.Fatalf("name = %q, want ok", out.Name)
	}
}
