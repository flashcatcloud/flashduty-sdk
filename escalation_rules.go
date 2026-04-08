package flashduty

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/sync/errgroup"
)

// rawEscalationRule represents the raw API response structure for escalation rules
type rawEscalationRule struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Description string `json:"description,omitempty"`
	ChannelID   int64  `json:"channel_id"`
	Status      string `json:"status"`
	Priority    int    `json:"priority"`
	AggrWindow  int    `json:"aggr_window"`
	Layers      []struct {
		MaxTimes       int     `json:"max_times"`
		NotifyStep     float64 `json:"notify_step"`
		EscalateWindow int     `json:"escalate_window"`
		ForceEscalate  bool    `json:"force_escalate"`
		Target         *struct {
			PersonIDs         []int64           `json:"person_ids,omitempty"`
			TeamIDs           []int64           `json:"team_ids,omitempty"`
			ScheduleToRoleIDs map[int64][]int64 `json:"schedule_to_role_ids,omitempty"`
			By                *struct {
				FollowPreference bool     `json:"follow_preference"`
				Critical         []string `json:"critical,omitempty"`
				Warning          []string `json:"warning,omitempty"`
				Info             []string `json:"info,omitempty"`
			} `json:"by,omitempty"`
			Webhooks []struct {
				Type     string         `json:"type"`
				Settings map[string]any `json:"settings,omitempty"`
			} `json:"webhooks,omitempty"`
		} `json:"target,omitempty"`
	} `json:"layers,omitempty"`
	TimeFilters []struct {
		Start  string `json:"start"`
		End    string `json:"end"`
		Repeat []int  `json:"repeat,omitempty"`
		CalID  string `json:"cal_id,omitempty"`
		IsOff  bool   `json:"is_off,omitempty"`
	} `json:"time_filters,omitempty"`
	Filters [][]struct {
		Key  string   `json:"key"`
		Oper string   `json:"oper"`
		Vals []string `json:"vals"`
	} `json:"filters,omitempty"`
}

// ListEscalationRulesOutput contains the result of listing escalation rules
type ListEscalationRulesOutput struct {
	Rules []EscalationRule `json:"rules"`
	Total int              `json:"total"`
}

// ListEscalationRules queries escalation rules for a channel
func (c *Client) ListEscalationRules(ctx context.Context, channelID int64) (*ListEscalationRulesOutput, error) {
	requestBody := map[string]any{
		"channel_id": channelID,
	}

	resp, err := c.makeRequest(ctx, "POST", "/channel/escalate/rule/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to query escalation rules: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []rawEscalationRule `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	rules := []EscalationRule{}
	if result.Data == nil || len(result.Data.Items) == 0 {
		return &ListEscalationRulesOutput{
			Rules: rules,
			Total: 0,
		}, nil
	}

	// Collect all IDs for enrichment
	personIDs := make([]int64, 0)
	teamIDs := make([]int64, 0)
	scheduleIDs := make([]int64, 0)

	for _, r := range result.Data.Items {
		for _, l := range r.Layers {
			if l.Target == nil {
				continue
			}
			for _, pid := range l.Target.PersonIDs {
				if pid != 0 {
					personIDs = append(personIDs, pid)
				}
			}
			for _, tid := range l.Target.TeamIDs {
				if tid != 0 {
					teamIDs = append(teamIDs, tid)
				}
			}
			for sid := range l.Target.ScheduleToRoleIDs {
				if sid != 0 {
					scheduleIDs = append(scheduleIDs, sid)
				}
			}
		}
	}

	// Fetch enrichment info concurrently (graceful degradation on errors)
	var personMap map[int64]PersonInfo
	var teamMap map[int64]TeamInfo
	var scheduleMap map[int64]ScheduleInfo
	var channelMap map[int64]ChannelInfo

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		personMap, err = c.fetchPersonInfos(gctx, personIDs)
		if err != nil {
			personMap = make(map[int64]PersonInfo)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		teamMap, err = c.fetchTeamInfos(gctx, teamIDs)
		if err != nil {
			teamMap = make(map[int64]TeamInfo)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		scheduleMap, err = c.fetchScheduleInfos(gctx, scheduleIDs)
		if err != nil {
			slog.Warn("failed to fetch schedule infos", "error", err, "schedule_ids", scheduleIDs)
			scheduleMap = make(map[int64]ScheduleInfo)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		channelMap, err = c.fetchChannelInfos(gctx, []int64{channelID})
		if err != nil {
			channelMap = make(map[int64]ChannelInfo)
		}
		return nil
	})

	_ = g.Wait()

	// Build enriched rules
	for _, r := range result.Data.Items {
		rule := EscalationRule{
			RuleID:      r.RuleID,
			RuleName:    r.RuleName,
			Description: r.Description,
			ChannelID:   r.ChannelID,
			Status:      r.Status,
			Priority:    r.Priority,
			AggrWindow:  r.AggrWindow,
		}

		if ch, ok := channelMap[r.ChannelID]; ok {
			rule.ChannelName = ch.ChannelName
		}

		if len(r.TimeFilters) > 0 {
			rule.TimeFilters = make([]TimeFilter, 0, len(r.TimeFilters))
			for _, tf := range r.TimeFilters {
				rule.TimeFilters = append(rule.TimeFilters, TimeFilter{
					Start:  tf.Start,
					End:    tf.End,
					Repeat: tf.Repeat,
					CalID:  tf.CalID,
					IsOff:  tf.IsOff,
				})
			}
		}

		if len(r.Filters) > 0 {
			rule.Filters = make(AlertFilters, 0, len(r.Filters))
			for _, andGroup := range r.Filters {
				group := make(AlertFilterGroup, 0, len(andGroup))
				for _, cond := range andGroup {
					group = append(group, AlertCondition{
						Key:  cond.Key,
						Oper: cond.Oper,
						Vals: cond.Vals,
					})
				}
				rule.Filters = append(rule.Filters, group)
			}
		}

		if len(r.Layers) > 0 {
			rule.Layers = make([]EscalationLayer, 0, len(r.Layers))
			for idx, l := range r.Layers {
				layer := EscalationLayer{
					LayerIdx:       idx,
					Timeout:        l.EscalateWindow,
					NotifyInterval: l.NotifyStep,
					MaxTimes:       l.MaxTimes,
					ForceEscalate:  l.ForceEscalate,
				}

				if l.Target != nil {
					target := &EscalationTarget{}

					if len(l.Target.PersonIDs) > 0 {
						target.Persons = make([]PersonTarget, 0, len(l.Target.PersonIDs))
						for _, pid := range l.Target.PersonIDs {
							pt := PersonTarget{PersonID: pid}
							if p, ok := personMap[pid]; ok {
								pt.PersonName = p.PersonName
								pt.Email = p.Email
							}
							target.Persons = append(target.Persons, pt)
						}
					}

					if len(l.Target.TeamIDs) > 0 {
						target.Teams = make([]TeamTarget, 0, len(l.Target.TeamIDs))
						for _, tid := range l.Target.TeamIDs {
							tt := TeamTarget{TeamID: tid}
							if team, ok := teamMap[tid]; ok {
								tt.TeamName = team.TeamName
							}
							target.Teams = append(target.Teams, tt)
						}
					}

					if len(l.Target.ScheduleToRoleIDs) > 0 {
						target.Schedules = make([]ScheduleTarget, 0, len(l.Target.ScheduleToRoleIDs))
						for sid, roleIDs := range l.Target.ScheduleToRoleIDs {
							st := ScheduleTarget{
								ScheduleID: sid,
								RoleIDs:    roleIDs,
							}
							if s, ok := scheduleMap[sid]; ok {
								st.ScheduleName = s.ScheduleName
							}
							target.Schedules = append(target.Schedules, st)
						}
					}

					if l.Target.By != nil {
						target.NotifyBy = &NotifyBy{
							FollowPreference: l.Target.By.FollowPreference,
							Critical:         l.Target.By.Critical,
							Warning:          l.Target.By.Warning,
							Info:             l.Target.By.Info,
						}
					}

					if len(l.Target.Webhooks) > 0 {
						target.Webhooks = make([]WebhookConfig, 0, len(l.Target.Webhooks))
						for _, wh := range l.Target.Webhooks {
							whConfig := WebhookConfig{
								Type:     wh.Type,
								Settings: wh.Settings,
							}
							if wh.Settings != nil {
								if alias, ok := wh.Settings["alias"].(string); ok {
									whConfig.Alias = alias
								}
							}
							target.Webhooks = append(target.Webhooks, whConfig)
						}
					}

					layer.Target = target
				}

				rule.Layers = append(rule.Layers, layer)
			}
		}

		rules = append(rules, rule)
	}

	return &ListEscalationRulesOutput{
		Rules: rules,
		Total: len(rules),
	}, nil
}
