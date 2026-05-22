package flashduty

import (
	"context"
	"fmt"
)

// ListSchedulesWithSlotsInput contains parameters for listing schedules with computed slots
type ListSchedulesWithSlotsInput struct {
	Start      int64   // Required: Unix seconds, start of time range for slot computation
	End        int64   // Required: Unix seconds, end of time range (max 45-day window)
	TeamIDs    []int64 // Optional: filter by teams
	Query      string  // Optional: schedule name search keyword
	IsMyTeam   bool    // Optional: only schedules in the current user's teams
	IsMyManage bool    // Optional: only schedules created by the current user inside their teams
	Limit      int     // Max results (default 20)
	Page       int     // Page number (default 1)

	// Deprecated: use TeamIDs.
	TeamID int64
}

// ScheduleMember represents a role and the people assigned under it.
type ScheduleMember struct {
	RoleID    int64   `json:"role_id"`
	PersonIDs []int64 `json:"person_ids"`
}

// ScheduleGroup represents a rotating group inside a schedule layer.
type ScheduleGroup struct {
	GroupName string           `json:"group_name"`
	Name      string           `json:"name"`
	Members   []ScheduleMember `json:"members"`
	Start     int64            `json:"start"`
	End       int64            `json:"end"`
}

// ScheduleLayer represents a configured layer in a schedule.
type ScheduleLayer struct {
	AccountID             int64            `json:"account_id"`
	Name                  string           `json:"name"`
	ScheduleID            int64            `json:"schedule_id"`
	Hidden                int              `json:"hidden"`
	Mode                  int              `json:"mode"`
	Weight                int              `json:"weight"`
	Groups                []ScheduleGroup  `json:"groups"`
	RotationDuration      int64            `json:"rotation_duration"`
	HandoffTime           int64            `json:"handoff_time"`
	EnableTime            int64            `json:"enable_time"`
	ExpireTime            int64            `json:"expire_time"`
	RestrictMode          int              `json:"restrict_mode"`
	RestrictStart         int64            `json:"restrict_start"`
	RestrictEnd           int64            `json:"restrict_end"`
	RestrictPeriods       []map[string]any `json:"restrict_periods,omitempty"`
	DayMask               map[string]any   `json:"day_mask,omitempty"`
	CreateAt              int64            `json:"create_at"`
	CreateBy              int64            `json:"create_by"`
	UpdateAt              int64            `json:"update_at"`
	UpdateBy              int64            `json:"update_by"`
	LayerName             string           `json:"layer_name,omitempty"`
	FairRotation          bool             `json:"fair_rotation,omitempty"`
	LayerStart            int64            `json:"layer_start,omitempty"`
	LayerEnd              *int64           `json:"layer_end,omitempty"`
	RotationUnit          string           `json:"rotation_unit,omitempty"`
	RotationValue         int64            `json:"rotation_value,omitempty"`
	MaskContinuousEnabled bool             `json:"mask_continuous_enabled,omitempty"`
}

// ScheduleCalculatedSchedule represents a computed slot inside a layer.
type ScheduleCalculatedSchedule struct {
	Start int64         `json:"start"`
	End   int64         `json:"end"`
	Group ScheduleGroup `json:"group"`
	Index int           `json:"index"`
}

// ScheduleCalculatedLayer represents computed schedule slots for a single layer.
type ScheduleCalculatedLayer struct {
	LayerName string                       `json:"layer_name"`
	Name      string                       `json:"name"`
	Mode      int                          `json:"mode"`
	Schedules []ScheduleCalculatedSchedule `json:"schedules"`
}

// ScheduleNotifyWebhook represents a configured schedule notification target.
type ScheduleNotifyWebhook struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Token string `json:"token,omitempty"`
	URL   string `json:"url,omitempty"`
}

// ScheduleNotify represents schedule notification settings.
type ScheduleNotify struct {
	AdvanceInTime int64                   `json:"advance_in_time,omitempty"`
	FixedTime     map[string]any          `json:"fixed_time,omitempty"`
	By            map[string]any          `json:"by,omitempty"`
	IM            map[string]string       `json:"im,omitempty"`
	Webhooks      []ScheduleNotifyWebhook `json:"webhooks,omitempty"`
}

// ScheduleOncallGroup represents the current or next on-call group snapshot.
type ScheduleOncallGroup struct {
	Start    int64         `json:"start"`
	End      int64         `json:"end"`
	Group    ScheduleGroup `json:"group"`
	UpdateAt int64         `json:"update_at"`
	Weight   int           `json:"weight"`
	Index    int           `json:"index"`
}

// ScheduleDetail represents the schedule payload returned by /schedule/list and /schedule/info.
type ScheduleDetail struct {
	ID             *int64                    `json:"id,omitempty"`
	Name           *string                   `json:"name,omitempty"`
	AccountID      int64                     `json:"account_id"`
	GroupID        *int64                    `json:"group_id,omitempty"`
	Disabled       *int                      `json:"disabled,omitempty"`
	CreateAt       int64                     `json:"create_at"`
	CreateBy       int64                     `json:"create_by"`
	UpdateAt       int64                     `json:"update_at"`
	UpdateBy       int64                     `json:"update_by"`
	Layers         []ScheduleLayer           `json:"layers,omitempty"`
	Field          string                    `json:"field,omitempty"`
	ScheduleLayers []ScheduleCalculatedLayer `json:"schedule_layers,omitempty"`
	FinalSchedule  ScheduleCalculatedLayer   `json:"final_schedule"`
	Start          int64                     `json:"start,omitempty"`
	End            int64                     `json:"end,omitempty"`
	Notify         ScheduleNotify            `json:"notify,omitempty"`
	ScheduleID     int64                     `json:"schedule_id"`
	ScheduleName   *string                   `json:"schedule_name,omitempty"`
	TeamID         *int64                    `json:"team_id,omitempty"`
	Description    *string                   `json:"description,omitempty"`
	LayerSchedules []ScheduleCalculatedLayer `json:"layer_schedules,omitempty"`
	Status         *int                      `json:"status,omitempty"`
	CurOncall      *ScheduleOncallGroup      `json:"cur_oncall,omitempty"`
	NextOncall     *ScheduleOncallGroup      `json:"next_oncall,omitempty"`
}

// ListSchedulesWithSlotsOutput contains schedules with computed on-call slots
type ListSchedulesWithSlotsOutput struct {
	Schedules []ScheduleDetail `json:"schedules"`
	Total     int64            `json:"total"`
}

// ListSchedulesWithSlots queries schedules with computed on-call slots for a time range
func (c *Client) ListSchedulesWithSlots(ctx context.Context, input *ListSchedulesWithSlotsInput) (*ListSchedulesWithSlotsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("list schedules input is required")
	}
	if input.Start <= 0 || input.End <= 0 {
		return nil, fmt.Errorf("start and end are required")
	}
	if input.IsMyTeam && input.IsMyManage {
		return nil, fmt.Errorf("is_my_team and is_my_manage cannot both be true")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}

	requestBody := map[string]any{
		"start": input.Start,
		"end":   input.End,
		"limit": limit,
		"p":     page,
	}
	if input.Query != "" {
		requestBody["query"] = input.Query
	}
	if input.IsMyTeam {
		requestBody["is_my_team"] = true
	}
	if input.IsMyManage {
		requestBody["is_my_manage"] = true
	}
	teamIDs := input.TeamIDs
	if len(teamIDs) == 0 && input.TeamID > 0 {
		teamIDs = []int64{input.TeamID}
	}
	if len(teamIDs) > 0 {
		requestBody["team_ids"] = teamIDs
	}

	result, err := postData[struct {
		Items []ScheduleDetail `json:"items"`
		Total int64            `json:"total"`
	}](c, ctx, "/schedule/list", requestBody, "failed to list schedules")
	if err != nil {
		return nil, err
	}

	schedules := []ScheduleDetail{}
	total := int64(0)
	if result != nil {
		schedules = result.Items
		total = result.Total
	}

	return &ListSchedulesWithSlotsOutput{
		Schedules: schedules,
		Total:     total,
	}, nil
}

// GetScheduleDetailInput contains parameters for getting schedule detail
type GetScheduleDetailInput struct {
	ScheduleID int64 // Required
	Start      int64 // Required: Unix seconds, start of time range
	End        int64 // Required: Unix seconds, end of time range
}

// GetScheduleDetailOutput contains full schedule detail
type GetScheduleDetailOutput struct {
	Schedule ScheduleDetail `json:"schedule"`
}

// GetScheduleDetail fetches detailed schedule information with computed slots
func (c *Client) GetScheduleDetail(ctx context.Context, input *GetScheduleDetailInput) (*GetScheduleDetailOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("get schedule detail input is required")
	}
	if input.ScheduleID <= 0 {
		return nil, fmt.Errorf("schedule_id is required")
	}
	if input.Start <= 0 || input.End <= 0 {
		return nil, fmt.Errorf("start and end are required")
	}

	requestBody := map[string]any{
		"schedule_id": input.ScheduleID,
		"start":       input.Start,
		"end":         input.End,
	}

	schedule, err := postOptionalData[ScheduleDetail](c, ctx, "/schedule/info", requestBody, "failed to get schedule detail")
	if err != nil {
		return nil, err
	}

	if schedule == nil {
		return nil, fmt.Errorf("schedule not found: %d", input.ScheduleID)
	}

	return &GetScheduleDetailOutput{Schedule: *schedule}, nil
}
