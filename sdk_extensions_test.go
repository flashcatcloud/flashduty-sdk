package flashduty

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSDKExtensionsTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	client, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	return client
}

func decodeJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return body
}

func TestQueryInsightByResponderDecodesMetricsBaseFields(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insight/responder" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if body["aggregate_unit"] != "day" {
			t.Fatalf("aggregate_unit = %#v, want day", body["aggregate_unit"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"responder_id":                      42,
						"responder_name":                    "Ada",
						"email":                             "ada@example.com",
						"ts":                                1713225600,
						"hours":                             "work",
						"total_incident_cnt":                3,
						"total_incidents_acknowledged":      2,
						"total_notifications":               9,
						"mean_seconds_to_ack":               15.5,
						"acknowledgement_pct":               0.66,
						"total_interruptions":               1,
						"total_engaged_seconds":             120,
						"total_incidents_escalated":         1,
						"total_seconds_to_ack":              31,
						"total_incidents_reassigned":        1,
						"total_incidents_timeout_escalated": 0,
					},
				},
			},
		})
	})

	out, err := client.QueryInsightByResponder(context.Background(), &InsightQueryInput{
		StartTime:     1713139200,
		EndTime:       1713744000,
		AggregateUnit: "day",
	})
	if err != nil {
		t.Fatalf("QueryInsightByResponder error: %v", err)
	}

	if len(out.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out.Items))
	}

	item := out.Items[0]
	if item.ResponderID != 42 || item.ResponderName != "Ada" {
		t.Fatalf("unexpected responder identity: %#v", item)
	}
	if item.TS != 1713225600 || item.Hours != "work" {
		t.Fatalf("unexpected bucket fields: %#v", item)
	}
	if item.TotalNotifications != 9 || item.TotalIncidentsAcknowledged != 2 {
		t.Fatalf("unexpected responder counters: %#v", item)
	}
}

func TestQueryInsightByTeamDecodesTimeBucketAndCloseMetrics(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insight/team" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"team_id":                         7,
						"team_name":                       "core",
						"ts":                              1713225600,
						"hours":                           "sleep",
						"total_incident_cnt":              5,
						"total_incidents_manually_closed": 2,
						"total_incidents_timeout_closed":  1,
						"total_seconds_to_close":          600,
					},
				},
			},
		})
	})

	out, err := client.QueryInsightByTeam(context.Background(), &InsightQueryInput{
		StartTime:  1713139200,
		EndTime:    1713744000,
		SplitHours: true,
	})
	if err != nil {
		t.Fatalf("QueryInsightByTeam error: %v", err)
	}

	item := out.Items[0]
	if item.TeamID != 7 || item.TeamName != "core" {
		t.Fatalf("unexpected team identity: %#v", item)
	}
	if item.TS != 1713225600 || item.Hours != "sleep" {
		t.Fatalf("unexpected time bucket: %#v", item)
	}
	if item.TotalIncidentsManuallyClosed != 2 || item.TotalIncidentsTimeoutClosed != 1 {
		t.Fatalf("unexpected close counters: %#v", item)
	}
}

func TestQueryInsightIncidentListDecodesCreatedAtAndCursor(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/insight/incident/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if _, ok := body["orderby"]; ok {
			t.Fatalf("unexpected orderby in request: %#v", body)
		}
		if body["search_after_ctx"] != "cursor-1" {
			t.Fatalf("search_after_ctx = %#v, want cursor-1", body["search_after_ctx"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"total":            11,
				"has_next_page":    true,
				"search_after_ctx": "cursor-2",
				"items": []any{
					map[string]any{
						"incident_id":      "507f1f77bcf86cd799439011",
						"title":            "db lag",
						"description":      "replica behind",
						"team_id":          10,
						"team_name":        "payments",
						"channel_id":       22,
						"channel_name":     "gateway",
						"progress":         "Closed",
						"severity":         "Critical",
						"created_at":       1713225600,
						"closed_by":        "manually",
						"seconds_to_ack":   60,
						"seconds_to_close": 900,
						"engaged_seconds":  300,
						"hours":            "off",
						"responders": []any{
							map[string]any{
								"person_id":       1,
								"person_name":     "Ada",
								"email":           "ada@example.com",
								"assigned_at":     1713225605,
								"acknowledged_at": 1713225660,
							},
						},
						"assigned_to": map[string]any{
							"person_ids":         []any{1, 2},
							"escalate_rule_id":   "507f1f77bcf86cd799439012",
							"escalate_rule_name": "primary",
							"layer_idx":          1,
							"type":               "assign",
							"assigned_at":        1713225605,
							"id":                 "asg-1",
						},
						"labels": map[string]any{
							"service": "gateway",
						},
						"fields": map[string]any{
							"cluster": "blue",
						},
						"notifications":       4,
						"interruptions":       2,
						"assignments":         1,
						"reassignments":       1,
						"acknowledgements":    1,
						"escalations":         1,
						"timeout_escalations": 0,
						"manual_escalations":  1,
						"creator_id":          99,
						"creator_name":        "Grace",
					},
				},
			},
		})
	})

	out, err := client.QueryInsightIncidentList(context.Background(), &QueryInsightIncidentListInput{
		InsightQueryInput: InsightQueryInput{
			StartTime: 1713139200,
			EndTime:   1713744000,
		},
		Page:           2,
		Limit:          5,
		SearchAfterCtx: "cursor-1",
		OrderBy:        "ignored",
	})
	if err != nil {
		t.Fatalf("QueryInsightIncidentList error: %v", err)
	}

	if !out.HasNextPage || out.SearchAfterCtx != "cursor-2" || out.Total != 11 {
		t.Fatalf("unexpected paging output: %#v", out)
	}
	item := out.Items[0]
	if item.CreatedAt != 1713225600 || item.TeamName != "payments" || item.CreatorName != "Grace" {
		t.Fatalf("unexpected incident item: %#v", item)
	}
	if len(item.Responders) != 1 || item.Responders[0].PersonName != "Ada" {
		t.Fatalf("unexpected responders: %#v", item.Responders)
	}
	if item.AssignedTo == nil || item.AssignedTo.EscalateRuleName != "primary" {
		t.Fatalf("unexpected assigned_to: %#v", item.AssignedTo)
	}
}

func TestGetIncidentFeedDecodesHasNextPageAndEnrichesNotifyPersons(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/incident/feed":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"has_next_page": true,
					"items": []any{
						map[string]any{
							"type":       "i_notify",
							"created_at": 1713225600,
							"creator_id": 7,
							"detail": map[string]any{
								"layer_idx": 1,
								"persons": []any{
									map[string]any{"person_id": 7},
									map[string]any{"person_id": 8},
								},
							},
						},
					},
				},
			})
		case "/person/infos":
			body := decodeJSONBody(t, r)
			personIDs, ok := body["person_ids"].([]any)
			if !ok || len(personIDs) != 2 {
				t.Fatalf("unexpected person_ids payload: %#v", body["person_ids"])
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"items": []any{
						map[string]any{"person_id": 7, "person_name": "Ada"},
						map[string]any{"person_id": 8, "person_name": "Grace"},
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	out, err := client.GetIncidentFeed(context.Background(), &GetIncidentFeedInput{
		IncidentID: "507f1f77bcf86cd799439011",
		Limit:      10,
		Page:       1,
	})
	if err != nil {
		t.Fatalf("GetIncidentFeed error: %v", err)
	}

	if !out.HasNextPage {
		t.Fatalf("expected has_next_page=true, got %#v", out)
	}
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 event, got %d", len(out.Items))
	}
	event := out.Items[0]
	if event.OperatorID != 7 || event.OperatorName != "Ada" {
		t.Fatalf("unexpected operator: %#v", event)
	}
	detail, ok := event.Detail.(map[string]any)
	if !ok {
		t.Fatalf("detail type = %T, want map[string]any", event.Detail)
	}
	persons, ok := detail["persons"].([]any)
	if !ok || len(persons) != 2 {
		t.Fatalf("unexpected enriched persons: %#v", detail["persons"])
	}
	firstPerson := persons[0].(map[string]any)
	if firstPerson["person_name"] != "Ada" {
		t.Fatalf("unexpected first person enrichment: %#v", firstPerson)
	}
}

func TestListPostMortemsUsesDocumentedFiltersAndResponseShape(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/post-mortem/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if _, ok := body["incident_id"]; ok {
			t.Fatalf("unexpected incident_id in request: %#v", body)
		}
		if body["status"] != "published" || body["order_by"] != "updated_at_seconds" {
			t.Fatalf("unexpected filters: %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"total":            2,
				"has_next_page":    true,
				"search_after_ctx": "pm-cursor-2",
				"items": []any{
					map[string]any{
						"account_id":         1,
						"post_mortem_id":     "pm-1",
						"template_id":        "tpl-1",
						"incident_ids":       []any{"507f1f77bcf86cd799439011"},
						"media_count":        2,
						"author_ids":         []any{7, 8},
						"team_id":            9,
						"channel_id":         11,
						"channel_name":       "gateway",
						"is_private":         true,
						"title":              "Gateway outage",
						"status":             "published",
						"created_at_seconds": 1713200000,
						"updated_at_seconds": 1713280000,
					},
				},
			},
		})
	})

	out, err := client.ListPostMortems(context.Background(), &ListPostMortemsInput{
		IncidentID:            "ignored",
		Status:                "published",
		TeamIDs:               []int64{9},
		ChannelIDs:            []int64{11},
		OrderBy:               "updated_at_seconds",
		Asc:                   true,
		SearchAfterCtx:        "pm-cursor-1",
		CreatedAtStartSeconds: 1713139200,
		CreatedAtEndSeconds:   1713744000,
	})
	if err != nil {
		t.Fatalf("ListPostMortems error: %v", err)
	}

	if !out.HasNextPage || out.SearchAfterCtx != "pm-cursor-2" || out.Total != 2 {
		t.Fatalf("unexpected pagination output: %#v", out)
	}
	if len(out.PostMortems) != 1 || out.PostMortems[0].CreatedAtSeconds != 1713200000 {
		t.Fatalf("unexpected post-mortems: %#v", out.PostMortems)
	}
}

func TestListAlertEventsUsesAlertIDOnly(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/alert/event/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if len(body) != 1 || body["alert_id"] != "507f1f77bcf86cd799439011" {
			t.Fatalf("unexpected request body: %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"event_id":         "507f1f77bcf86cd799439012",
						"alert_id":         "507f1f77bcf86cd799439011",
						"integration_id":   55,
						"integration_type": "prometheus",
						"title":            "cpu high",
						"description":      "sustained load",
						"event_severity":   "Critical",
						"event_status":     "Critical",
						"event_time":       1713225600,
						"created_at":       1713225600,
						"updated_at":       1713225660,
						"labels": map[string]any{
							"service": "gateway",
						},
					},
				},
			},
		})
	})

	out, err := client.ListAlertEvents(context.Background(), &ListAlertEventsInput{
		AlertID:   "507f1f77bcf86cd799439011",
		StartTime: 1,
		EndTime:   2,
		Limit:     50,
		Page:      3,
	})
	if err != nil {
		t.Fatalf("ListAlertEvents error: %v", err)
	}

	if len(out.AlertEvents) != 1 || out.AlertEvents[0].IntegrationType != "prometheus" {
		t.Fatalf("unexpected alert events: %#v", out.AlertEvents)
	}
}

func TestSearchAuditLogsUsesDocsAndCursor(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audit/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if body["person_id"] != float64(9) {
			t.Fatalf("unexpected person_id: %#v", body["person_id"])
		}
		if body["search_after_ctx"] != "audit-cursor-1" {
			t.Fatalf("unexpected cursor: %#v", body["search_after_ctx"])
		}
		operations, ok := body["operations"].([]any)
		if !ok || len(operations) != 2 {
			t.Fatalf("unexpected operations: %#v", body["operations"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"total":            2,
				"search_after_ctx": "audit-cursor-2",
				"docs": []any{
					map[string]any{
						"created_at":     1713225600,
						"account_id":     1,
						"member_id":      9,
						"member_name":    "Ada",
						"request_id":     "req-1",
						"ip":             "127.0.0.1",
						"operation":      "schedule.update",
						"operation_name": "Update Schedule",
						"body":           `{"name":"primary"}`,
						"params": []any{
							map[string]any{"Key": "schedule_id", "Value": "5"},
						},
						"is_dangerous": true,
						"is_write":     true,
					},
				},
			},
		})
	})

	isDangerous := true
	isWrite := true
	out, err := client.SearchAuditLogs(context.Background(), &SearchAuditLogsInput{
		StartTime:      1713139200,
		EndTime:        1713744000,
		Limit:          10,
		SearchAfterCtx: "audit-cursor-1",
		Operations:     []string{"schedule.update", "escalation.update"},
		PersonID:       9,
		IsDangerous:    &isDangerous,
		IsWrite:        &isWrite,
	})
	if err != nil {
		t.Fatalf("SearchAuditLogs error: %v", err)
	}

	if out.Total != 2 || out.SearchAfterCtx != "audit-cursor-2" {
		t.Fatalf("unexpected audit paging: %#v", out)
	}
	if len(out.AuditLogs) != 1 || out.AuditLogs[0].MemberName != "Ada" {
		t.Fatalf("unexpected audit logs: %#v", out.AuditLogs)
	}
}

func TestListSchedulesWithSlotsUsesTeamIDsAndDecodesComputedLayers(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/schedule/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if _, ok := body["team_id"]; ok {
			t.Fatalf("unexpected team_id in request: %#v", body)
		}
		teamIDs, ok := body["team_ids"].([]any)
		if !ok || len(teamIDs) != 2 {
			t.Fatalf("unexpected team_ids: %#v", body["team_ids"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"total": 1,
				"items": []any{
					map[string]any{
						"id":          5,
						"name":        "primary",
						"account_id":  1,
						"create_at":   1,
						"create_by":   2,
						"update_at":   3,
						"update_by":   4,
						"schedule_id": 5,
						"final_schedule": map[string]any{
							"layer_name": "final",
							"name":       "final",
							"mode":       1,
							"schedules": []any{
								map[string]any{
									"start": 1713225600,
									"end":   1713232800,
									"index": 0,
									"group": map[string]any{
										"group_name": "day",
										"name":       "day",
										"start":      1713225600,
										"end":        1713232800,
										"members": []any{
											map[string]any{
												"role_id":    1,
												"person_ids": []any{7, 8},
											},
										},
									},
								},
							},
						},
						"cur_oncall": map[string]any{
							"start":  1713225600,
							"end":    1713232800,
							"weight": 1,
							"index":  0,
							"group": map[string]any{
								"group_name": "day",
								"name":       "day",
								"members": []any{
									map[string]any{
										"role_id":    1,
										"person_ids": []any{7, 8},
									},
								},
							},
						},
						"next_oncall": map[string]any{
							"start":  1713232800,
							"end":    1713240000,
							"weight": 1,
							"index":  1,
							"group": map[string]any{
								"group_name": "night",
								"name":       "night",
								"members": []any{
									map[string]any{
										"role_id":    1,
										"person_ids": []any{9},
									},
								},
							},
						},
					},
				},
			},
		})
	})

	out, err := client.ListSchedulesWithSlots(context.Background(), &ListSchedulesWithSlotsInput{
		Start:   1713139200,
		End:     1713744000,
		TeamIDs: []int64{1, 2},
		Query:   "primary",
	})
	if err != nil {
		t.Fatalf("ListSchedulesWithSlots error: %v", err)
	}

	if out.Total != 1 || len(out.Schedules) != 1 {
		t.Fatalf("unexpected schedules output: %#v", out)
	}
	schedule := out.Schedules[0]
	if schedule.FinalSchedule.LayerName != "final" {
		t.Fatalf("unexpected final schedule: %#v", schedule.FinalSchedule)
	}
	if schedule.CurOncall == nil || len(schedule.CurOncall.Group.Members) != 1 {
		t.Fatalf("unexpected current oncall: %#v", schedule.CurOncall)
	}
}

func TestGetScheduleDetailDecodesOncallSnapshot(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/schedule/info" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"schedule_id": 5,
				"account_id":  1,
				"create_at":   1,
				"create_by":   2,
				"update_at":   3,
				"update_by":   4,
				"final_schedule": map[string]any{
					"layer_name": "final",
					"name":       "final",
					"mode":       1,
					"schedules":  []any{},
				},
				"cur_oncall": map[string]any{
					"start": 1713225600,
					"end":   1713232800,
					"group": map[string]any{
						"group_name": "day",
						"name":       "day",
						"members": []any{
							map[string]any{
								"role_id":    1,
								"person_ids": []any{7},
							},
						},
					},
				},
			},
		})
	})

	out, err := client.GetScheduleDetail(context.Background(), &GetScheduleDetailInput{
		ScheduleID: 5,
		Start:      1713139200,
		End:        1713744000,
	})
	if err != nil {
		t.Fatalf("GetScheduleDetail error: %v", err)
	}

	if out.Schedule.CurOncall == nil || out.Schedule.CurOncall.Group.GroupName != "day" {
		t.Fatalf("unexpected schedule detail: %#v", out.Schedule)
	}
}

func TestQueryNotificationTrendPreservesAllCounters(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/report/oncall/notifications" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"ts":        1713225600,
						"sms_cnt":   2,
						"voice_cnt": 3,
						"email_cnt": 4,
					},
				},
			},
		})
	})

	out, err := client.QueryNotificationTrend(context.Background(), &QueryNotificationTrendInput{
		ChannelIDs: []int64{1},
		Step:       "day",
		StartTime:  1713139200,
		EndTime:    1713744000,
	})
	if err != nil {
		t.Fatalf("QueryNotificationTrend error: %v", err)
	}

	point := out.DataPoints[0]
	if point.Timestamp != 1713225600 || point.SMSCount != 2 || point.VoiceCount != 3 || point.EmailCount != 4 {
		t.Fatalf("unexpected notification trend point: %#v", point)
	}
}

func TestQueryChangeTrendPreservesAllCounters(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/report/oncall/changes" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []any{
					map[string]any{
						"ts":               1713225600,
						"change_cnt":       5,
						"change_event_cnt": 8,
					},
				},
			},
		})
	})

	out, err := client.QueryChangeTrend(context.Background(), &QueryChangeTrendInput{
		Step:      "day",
		StartTime: 1713139200,
		EndTime:   1713744000,
	})
	if err != nil {
		t.Fatalf("QueryChangeTrend error: %v", err)
	}

	point := out.DataPoints[0]
	if point.Timestamp != 1713225600 || point.ChangeCount != 5 || point.ChangeEventCount != 8 {
		t.Fatalf("unexpected change trend point: %#v", point)
	}
}

func TestQueryMonitorRuleStatusDecodesFolderRows(t *testing.T) {
	client := newSDKExtensionsTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/monit/rule/counter/status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body := decodeJSONBody(t, r)
		if len(body) != 0 {
			t.Fatalf("unexpected request body: %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"folder_id":            1,
					"folder_name":          "core",
					"rule_total":           12,
					"triggered_rule_count": 3,
				},
			},
		})
	})

	out, err := client.QueryMonitorRuleStatus(context.Background(), nil)
	if err != nil {
		t.Fatalf("QueryMonitorRuleStatus error: %v", err)
	}

	if len(out.Statuses) != 1 || out.Statuses[0].FolderName != "core" {
		t.Fatalf("unexpected monitor statuses: %#v", out.Statuses)
	}
}
