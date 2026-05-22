package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestUnackIncidentsPostsIncidentIDs(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/unack" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeJSONBody(t, r)
		if !reflect.DeepEqual(body["incident_ids"], []any{"inc-1", "inc-2"}) {
			t.Fatalf("incident_ids = %#v, want inc-1/inc-2", body["incident_ids"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})

	if err := client.UnackIncidents(context.Background(), []string{"inc-1", "inc-2"}); err != nil {
		t.Fatalf("UnackIncidents error: %v", err)
	}
}

func TestCommentIncidentsIncludesNotifyOptions(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/comment" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeJSONBody(t, r)
		if body["comment"] != "investigating" {
			t.Fatalf("comment = %#v, want investigating", body["comment"])
		}
		notify, ok := body["notify"].(map[string]any)
		if !ok {
			t.Fatalf("notify missing or wrong type: %#v", body["notify"])
		}
		if notify["follow_preference"] != true || notify["template_id"] != "tpl-1" {
			t.Fatalf("unexpected notify: %#v", notify)
		}
		if !reflect.DeepEqual(notify["personal_channels"], []any{"email", "sms"}) {
			t.Fatalf("personal_channels = %#v", notify["personal_channels"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})

	err := client.CommentIncidents(context.Background(), &IncidentCommentInput{
		IncidentIDs: []string{"inc-1"},
		Comment:     "investigating",
		MuteReply:   true,
		Notify: &IncidentNotifyInput{
			FollowPreference: true,
			PersonalChannels: []string{"email", "sms"},
			TemplateID:       "tpl-1",
		},
	})
	if err != nil {
		t.Fatalf("CommentIncidents error: %v", err)
	}
}

func TestCreateIncidentWarRoomIncludesObserversAndDecodesChat(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/war-room/create" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeJSONBody(t, r)
		if body["incident_id"] != "inc-1" || body["integration_id"] != float64(42) {
			t.Fatalf("unexpected request body: %#v", body)
		}
		if body["add_observers"] != true {
			t.Fatalf("add_observers = %#v, want true", body["add_observers"])
		}
		if !reflect.DeepEqual(body["member_ids"], []any{float64(101), float64(202)}) {
			t.Fatalf("member_ids = %#v", body["member_ids"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"chat_id":    "chat-1",
				"chat_name":  "INC outage",
				"share_link": "https://chat.example/1",
			},
		})
	})

	out, err := client.CreateIncidentWarRoom(context.Background(), &IncidentWarRoomCreateInput{
		IncidentID:    "inc-1",
		IntegrationID: 42,
		MemberIDs:     []int64{101, 202},
		AddObservers:  true,
	})
	if err != nil {
		t.Fatalf("CreateIncidentWarRoom error: %v", err)
	}
	if out.ChatID != "chat-1" || out.ShareLink != "https://chat.example/1" {
		t.Fatalf("unexpected war room: %#v", out)
	}
}

func TestListIncidentWarRoomsDecodesItems(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/war-room/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeJSONBody(t, r)
		if body["incident_id"] != "inc-1" {
			t.Fatalf("incident_id = %#v, want inc-1", body["incident_id"])
		}
		if _, ok := body["integration_id"]; ok {
			t.Fatalf("unexpected integration_id in request: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"integration_id": 42,
						"chat_id":        "chat-1",
						"chat_name":      "INC outage",
						"incident_id":    "inc-1",
						"status":         "enabled",
						"plugin_type":    "feishu_app",
					},
				},
			},
		})
	})

	out, err := client.ListIncidentWarRooms(context.Background(), &IncidentWarRoomListInput{IncidentID: "inc-1"})
	if err != nil {
		t.Fatalf("ListIncidentWarRooms error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].IntegrationID != 42 || out.Items[0].ChatID != "chat-1" {
		t.Fatalf("unexpected war room list: %#v", out)
	}
}

func TestListWarRoomEnabledDataSourcesDecodesItems(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasource/im/war-room-enabled/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := decodeJSONBody(t, r)
		if len(body) != 0 {
			t.Fatalf("expected empty request body, got %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"data_source_id": 42,
						"name":           "Feishu",
						"plugin_type":    "feishu_app",
						"category":       "im",
						"settings": map[string]any{
							"war_room_enabled": true,
						},
					},
				},
			},
		})
	})

	out, err := client.ListWarRoomEnabledDataSources(context.Background())
	if err != nil {
		t.Fatalf("ListWarRoomEnabledDataSources error: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].DataSourceID != 42 || out.Items[0].PluginType != "feishu_app" {
		t.Fatalf("unexpected datasource list: %#v", out)
	}
}
