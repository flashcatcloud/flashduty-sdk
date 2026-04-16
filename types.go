package flashduty

// EnrichedIncident contains full incident data with human-readable names
type EnrichedIncident struct {
	// Basic fields
	IncidentID  string `json:"incident_id" toon:"incident_id"`
	Title       string `json:"title" toon:"title"`
	Description string `json:"description,omitempty" toon:"description,omitempty"`
	Severity    string `json:"severity" toon:"severity"`
	Progress    string `json:"progress" toon:"progress"`

	// Time fields
	StartTime int64 `json:"start_time" toon:"start_time"`
	AckTime   int64 `json:"ack_time,omitempty" toon:"ack_time,omitempty"`
	CloseTime int64 `json:"close_time,omitempty" toon:"close_time,omitempty"`

	// Channel (enriched)
	ChannelID   int64  `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName string `json:"channel_name,omitempty" toon:"channel_name,omitempty"`

	// Creator (enriched)
	CreatorID    int64  `json:"creator_id,omitempty" toon:"creator_id,omitempty"`
	CreatorName  string `json:"creator_name,omitempty" toon:"creator_name,omitempty"`
	CreatorEmail string `json:"creator_email,omitempty" toon:"creator_email,omitempty"`

	// Closer (enriched)
	CloserID   int64  `json:"closer_id,omitempty" toon:"closer_id,omitempty"`
	CloserName string `json:"closer_name,omitempty" toon:"closer_name,omitempty"`

	// Responders (enriched)
	Responders []EnrichedResponder `json:"responders,omitempty" toon:"responders,omitempty"`

	// Timeline (full)
	Timeline []TimelineEvent `json:"timeline,omitempty" toon:"timeline,omitempty"`

	// Alerts (preview)
	AlertsPreview []AlertPreview `json:"alerts_preview,omitempty" toon:"alerts_preview,omitempty"`
	AlertsTotal   int            `json:"alerts_total" toon:"alerts_total"`

	// Other
	Labels       map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
	CustomFields map[string]any    `json:"custom_fields,omitempty" toon:"custom_fields,omitempty"`
}

// EnrichedResponder contains responder info with human-readable names
type EnrichedResponder struct {
	PersonID       int64  `json:"person_id" toon:"person_id"`
	PersonName     string `json:"person_name" toon:"person_name"`
	Email          string `json:"email,omitempty" toon:"email,omitempty"`
	AssignedAt     int64  `json:"assigned_at,omitempty" toon:"assigned_at,omitempty"`
	AcknowledgedAt int64  `json:"acknowledged_at,omitempty" toon:"acknowledged_at,omitempty"`
}

// TimelineEvent represents an entry in incident timeline
type TimelineEvent struct {
	Type         string `json:"type" toon:"type"`
	Timestamp    int64  `json:"timestamp" toon:"timestamp"`
	OperatorID   int64  `json:"operator_id,omitempty" toon:"operator_id,omitempty"`
	OperatorName string `json:"operator_name,omitempty" toon:"operator_name,omitempty"`
	Detail       any    `json:"detail,omitempty" toon:"detail,omitempty"`
}

// AlertPreview represents a preview of an alert
type AlertPreview struct {
	AlertID   string            `json:"alert_id" toon:"alert_id"`
	Title     string            `json:"title" toon:"title"`
	Severity  string            `json:"severity" toon:"severity"`
	Status    string            `json:"status" toon:"status"`
	StartTime int64             `json:"start_time" toon:"start_time"`
	Labels    map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
}

// PersonInfo represents person information from /person/infos API
type PersonInfo struct {
	PersonID   int64  `json:"person_id" toon:"person_id"`
	PersonName string `json:"person_name" toon:"person_name"`
	Email      string `json:"email,omitempty" toon:"email,omitempty"`
	Avatar     string `json:"avatar,omitempty" toon:"avatar,omitempty"`
	As         string `json:"as,omitempty" toon:"as,omitempty"`
}

// ChannelInfo represents channel information with enriched fields
type ChannelInfo struct {
	ChannelID   int64  `json:"channel_id" toon:"channel_id"`
	ChannelName string `json:"channel_name" toon:"channel_name"`
	TeamID      int64  `json:"team_id,omitempty" toon:"team_id,omitempty"`
	TeamName    string `json:"team_name,omitempty" toon:"team_name,omitempty"`
	CreatorID   int64  `json:"creator_id,omitempty" toon:"creator_id,omitempty"`
	CreatorName string `json:"creator_name,omitempty" toon:"creator_name,omitempty"`
}

// TeamInfo represents team information
type TeamInfo struct {
	TeamID   int64        `json:"team_id" toon:"team_id"`
	TeamName string       `json:"team_name" toon:"team_name"`
	Members  []TeamMember `json:"members,omitempty" toon:"members,omitempty"`
}

// TeamMember represents a team member
type TeamMember struct {
	PersonID   int64  `json:"person_id" toon:"person_id"`
	PersonName string `json:"person_name" toon:"person_name"`
	Email      string `json:"email,omitempty" toon:"email,omitempty"`
}

// FieldInfo represents custom field definition
type FieldInfo struct {
	FieldID      string   `json:"field_id" toon:"field_id"`
	FieldName    string   `json:"field_name" toon:"field_name"`
	DisplayName  string   `json:"display_name" toon:"display_name"`
	FieldType    string   `json:"field_type" toon:"field_type"`
	ValueType    string   `json:"value_type" toon:"value_type"`
	Options      []string `json:"options,omitempty" toon:"options,omitempty"`
	DefaultValue any      `json:"default_value,omitempty" toon:"default_value,omitempty"`
}

// EscalationRule represents an escalation rule with full details
type EscalationRule struct {
	RuleID      string            `json:"rule_id" toon:"rule_id"`
	RuleName    string            `json:"rule_name" toon:"rule_name"`
	Description string            `json:"description,omitempty" toon:"description,omitempty"`
	ChannelID   int64             `json:"channel_id" toon:"channel_id"`
	ChannelName string            `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	Status      string            `json:"status,omitempty" toon:"status,omitempty"`
	Priority    int               `json:"priority" toon:"priority"`
	AggrWindow  int               `json:"aggr_window" toon:"aggr_window"`
	Layers      []EscalationLayer `json:"layers,omitempty" toon:"layers,omitempty"`
	TimeFilters []TimeFilter      `json:"time_filters,omitempty" toon:"time_filters,omitempty"`
	Filters     AlertFilters      `json:"filters,omitempty" toon:"filters,omitempty"`
}

// EscalationLayer represents a layer in an escalation rule
type EscalationLayer struct {
	LayerIdx       int               `json:"layer_idx" toon:"layer_idx"`
	Timeout        int               `json:"timeout" toon:"timeout"`
	NotifyInterval float64           `json:"notify_interval,omitempty" toon:"notify_interval,omitempty"`
	MaxTimes       int               `json:"max_times,omitempty" toon:"max_times,omitempty"`
	ForceEscalate  bool              `json:"force_escalate,omitempty" toon:"force_escalate,omitempty"`
	Target         *EscalationTarget `json:"target,omitempty" toon:"target,omitempty"`
}

// EscalationTarget represents the complete target configuration for a layer
type EscalationTarget struct {
	Persons   []PersonTarget   `json:"persons,omitempty" toon:"persons,omitempty"`
	Teams     []TeamTarget     `json:"teams,omitempty" toon:"teams,omitempty"`
	Schedules []ScheduleTarget `json:"schedules,omitempty" toon:"schedules,omitempty"`
	NotifyBy  *NotifyBy        `json:"notify_by,omitempty" toon:"notify_by,omitempty"`
	Webhooks  []WebhookConfig  `json:"webhooks,omitempty" toon:"webhooks,omitempty"`
}

// PersonTarget represents a person in escalation target
type PersonTarget struct {
	PersonID   int64  `json:"person_id" toon:"person_id"`
	PersonName string `json:"person_name,omitempty" toon:"person_name,omitempty"`
	Email      string `json:"email,omitempty" toon:"email,omitempty"`
}

// TeamTarget represents a team in escalation target with members
type TeamTarget struct {
	TeamID   int64          `json:"team_id" toon:"team_id"`
	TeamName string         `json:"team_name,omitempty" toon:"team_name,omitempty"`
	Members  []PersonTarget `json:"members,omitempty" toon:"members,omitempty"`
}

// ScheduleTarget represents a schedule in escalation target
type ScheduleTarget struct {
	ScheduleID   int64   `json:"schedule_id" toon:"schedule_id"`
	ScheduleName string  `json:"schedule_name,omitempty" toon:"schedule_name,omitempty"`
	RoleIDs      []int64 `json:"role_ids,omitempty" toon:"role_ids,omitempty"`
}

// NotifyBy represents direct message notification configuration
type NotifyBy struct {
	FollowPreference bool     `json:"follow_preference" toon:"follow_preference"`
	Critical         []string `json:"critical,omitempty" toon:"critical,omitempty"`
	Warning          []string `json:"warning,omitempty" toon:"warning,omitempty"`
	Info             []string `json:"info,omitempty" toon:"info,omitempty"`
}

// WebhookConfig represents a webhook configuration in escalation target
type WebhookConfig struct {
	Type     string         `json:"type" toon:"type"`
	Alias    string         `json:"alias,omitempty" toon:"alias,omitempty"`
	Settings map[string]any `json:"settings,omitempty" toon:"settings,omitempty"`
}

// TimeFilter represents time-based filter for rule activation
type TimeFilter struct {
	Start  string `json:"start" toon:"start"`
	End    string `json:"end" toon:"end"`
	Repeat []int  `json:"repeat,omitempty" toon:"repeat,omitempty"`
	CalID  string `json:"cal_id,omitempty" toon:"cal_id,omitempty"`
	IsOff  bool   `json:"is_off,omitempty" toon:"is_off,omitempty"`
}

// AlertFilters represents alert filter conditions as OR groups of AND conditions
type AlertFilters []AlertFilterGroup

// AlertFilterGroup represents AND conditions within an OR group
type AlertFilterGroup []AlertCondition

// AlertCondition represents a single filter condition
type AlertCondition struct {
	Key  string   `json:"key" toon:"key"`
	Oper string   `json:"oper" toon:"oper"`
	Vals []string `json:"vals" toon:"vals"`
}

// ScheduleInfo represents schedule information from /schedule/infos API
type ScheduleInfo struct {
	ScheduleID   int64  `json:"schedule_id" toon:"schedule_id"`
	ScheduleName string `json:"schedule_name" toon:"schedule_name"`
}

// StatusPage represents a status page
type StatusPage struct {
	PageID        int64             `json:"page_id" toon:"page_id"`
	PageName      string            `json:"page_name" toon:"page_name"`
	Slug          string            `json:"slug,omitempty" toon:"slug,omitempty"`
	Description   string            `json:"description,omitempty" toon:"description,omitempty"`
	Sections      []StatusSection   `json:"sections,omitempty" toon:"sections,omitempty"`
	Components    []StatusComponent `json:"components,omitempty" toon:"components,omitempty"`
	OverallStatus string            `json:"overall_status,omitempty" toon:"overall_status,omitempty"`
}

// StatusSection represents a section in status page
type StatusSection struct {
	SectionID   string `json:"section_id" toon:"section_id"`
	SectionName string `json:"section_name" toon:"section_name"`
}

// StatusComponent represents a component in status page
type StatusComponent struct {
	ComponentID   string `json:"component_id" toon:"component_id"`
	ComponentName string `json:"component_name" toon:"component_name"`
	Status        string `json:"status" toon:"status"`
	SectionID     string `json:"section_id,omitempty" toon:"section_id,omitempty"`
}

// StatusChange represents a change event on status page
type StatusChange struct {
	ChangeID    int64            `json:"change_id" toon:"change_id"`
	PageID      int64            `json:"page_id" toon:"page_id"`
	Title       string           `json:"title" toon:"title"`
	Description string           `json:"description,omitempty" toon:"description,omitempty"`
	Type        string           `json:"type" toon:"type"`
	Status      string           `json:"status" toon:"status"`
	CreatedAt   int64            `json:"created_at" toon:"created_at"`
	UpdatedAt   int64            `json:"updated_at,omitempty" toon:"updated_at,omitempty"`
	Timelines   []ChangeTimeline `json:"timelines,omitempty" toon:"timelines,omitempty"`
}

// ChangeTimeline represents a timeline entry in status change
type ChangeTimeline struct {
	TimelineID  int64  `json:"timeline_id" toon:"timeline_id"`
	At          int64  `json:"at" toon:"at"`
	Status      string `json:"status,omitempty" toon:"status,omitempty"`
	Description string `json:"description,omitempty" toon:"description,omitempty"`
}

// Change represents a change record
type Change struct {
	ChangeID    string            `json:"change_id" toon:"change_id"`
	Title       string            `json:"title" toon:"title"`
	Description string            `json:"description,omitempty" toon:"description,omitempty"`
	Type        string            `json:"type,omitempty" toon:"type,omitempty"`
	Status      string            `json:"status,omitempty" toon:"status,omitempty"`
	ChannelID   int64             `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName string            `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	CreatorID   int64             `json:"creator_id,omitempty" toon:"creator_id,omitempty"`
	CreatorName string            `json:"creator_name,omitempty" toon:"creator_name,omitempty"`
	StartTime   int64             `json:"start_time,omitempty" toon:"start_time,omitempty"`
	EndTime     int64             `json:"end_time,omitempty" toon:"end_time,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
}

// RawTimelineItem represents raw timeline data from API
type RawTimelineItem struct {
	RefID     string         `json:"ref_id,omitempty"`
	Type      string         `json:"type"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at,omitempty"`
	AccountID int64          `json:"account_id,omitempty"`
	CreatorID int64          `json:"creator_id,omitempty"`
	PersonID  int64          `json:"person_id,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
}

// RawIncident represents raw incident data from API
type RawIncident struct {
	IncidentID  string            `json:"incident_id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Severity    string            `json:"incident_severity"`
	Progress    string            `json:"progress"`
	StartTime   int64             `json:"start_time"`
	AckTime     int64             `json:"ack_time,omitempty"`
	CloseTime   int64             `json:"close_time,omitempty"`
	ChannelID   int64             `json:"channel_id,omitempty"`
	CreatorID   int64             `json:"creator_id,omitempty"`
	CloserID    int64             `json:"closer_id,omitempty"`
	Responders  []RawResponder    `json:"responders,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Fields      map[string]any    `json:"fields,omitempty"`
}

// RawResponder represents raw responder data from API
type RawResponder struct {
	PersonID       int64  `json:"person_id"`
	AssignedAt     int64  `json:"assigned_at,omitempty"`
	AcknowledgedAt int64  `json:"acknowledged_at,omitempty"`
	PersonName     string `json:"person_name,omitempty"`
	Email          string `json:"email,omitempty"`
	As             string `json:"as,omitempty"`
}

// AssignedTo represents the current assignment target for an incident.
type AssignedTo struct {
	PersonIDs        []int64  `json:"person_ids,omitempty" toon:"person_ids,omitempty"`
	EscalateRuleID   string   `json:"escalate_rule_id,omitempty" toon:"escalate_rule_id,omitempty"`
	LayerIdx         int      `json:"layer_idx,omitempty" toon:"layer_idx,omitempty"`
	Type             string   `json:"type,omitempty" toon:"type,omitempty"`
	Emails           []string `json:"emails,omitempty" toon:"emails,omitempty"`
	EscalateRuleName string   `json:"escalate_rule_name,omitempty" toon:"escalate_rule_name,omitempty"`
	AssignedAt       int64    `json:"assigned_at,omitempty" toon:"assigned_at,omitempty"`
	ID               string   `json:"id,omitempty" toon:"id,omitempty"`
}

// MemberListResponse represents the response for member list API
type MemberListResponse struct {
	Error *DutyError `json:"error,omitempty"`
	Data  *struct {
		P     int          `json:"p"`
		Limit int          `json:"limit"`
		Total int          `json:"total"`
		Items []MemberItem `json:"items"`
	} `json:"data,omitempty"`
}

// MemberItem represents a member item as defined in the OpenAPI spec
type MemberItem struct {
	MemberID       int    `json:"member_id"`
	MemberName     string `json:"member_name"`
	Phone          string `json:"phone,omitempty"`
	PhoneVerified  bool   `json:"phone_verified,omitempty"`
	Email          string `json:"email,omitempty"`
	EmailVerified  bool   `json:"email_verified,omitempty"`
	AccountRoleIDs []int  `json:"account_role_ids,omitempty"`
	TimeZone       string `json:"time_zone,omitempty"`
	Locale         string `json:"locale,omitempty"`
	Status         string `json:"status"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
	RefID          string `json:"ref_id,omitempty"`
}

// MemberItemShort represents a short member item for invite response
type MemberItemShort struct {
	MemberID   int    `json:"MemberID"`
	MemberName string `json:"MemberName"`
}

// TemplateVariable describes a variable available in notification templates
type TemplateVariable struct {
	Name        string `json:"name" toon:"name"`
	Type        string `json:"type" toon:"type"`
	Description string `json:"description" toon:"description"`
	Example     string `json:"example,omitempty" toon:"example,omitempty"`
	Category    string `json:"category" toon:"category"`
}

// TemplateFunction describes a function available in notification templates
type TemplateFunction struct {
	Name        string `json:"name" toon:"name"`
	Syntax      string `json:"syntax" toon:"syntax"`
	Description string `json:"description" toon:"description"`
}

// MetricsBase represents the shared bucket and dimension fields returned by /insight/* APIs.
type MetricsBase struct {
	Hours         string `json:"hours,omitempty" toon:"hours,omitempty"`
	TS            int64  `json:"ts,omitempty" toon:"ts,omitempty"`
	ChannelID     int64  `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	TeamID        int64  `json:"team_id,omitempty" toon:"team_id,omitempty"`
	ResponderID   int64  `json:"responder_id,omitempty" toon:"responder_id,omitempty"`
	AccountID     int64  `json:"account_id,omitempty" toon:"account_id,omitempty"`
	TeamName      string `json:"team_name,omitempty" toon:"team_name,omitempty"`
	ChannelName   string `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	ResponderName string `json:"responder_name,omitempty" toon:"responder_name,omitempty"`
}

// DimensionInsightItem represents pre-aggregated metrics for a team or channel dimension.
type DimensionInsightItem struct {
	MetricsBase

	TotalIncidentCnt                int     `json:"total_incident_cnt" toon:"total_incident_cnt"`
	TotalIncidentsAcknowledged      int     `json:"total_incidents_acknowledged" toon:"total_incidents_acknowledged"`
	TotalIncidentsClosed            int     `json:"total_incidents_closed" toon:"total_incidents_closed"`
	TotalIncidentsAutoClosed        int     `json:"total_incidents_auto_closed" toon:"total_incidents_auto_closed"`
	TotalIncidentsManuallyClosed    int     `json:"total_incidents_manually_closed" toon:"total_incidents_manually_closed"`
	TotalIncidentsTimeoutClosed     int     `json:"total_incidents_timeout_closed" toon:"total_incidents_timeout_closed"`
	TotalIncidentsEscalated         int     `json:"total_incidents_escalated" toon:"total_incidents_escalated"`
	TotalIncidentsManuallyEscalated int     `json:"total_incidents_manually_escalated" toon:"total_incidents_manually_escalated"`
	TotalIncidentsTimeoutEscalated  int     `json:"total_incidents_timeout_escalated" toon:"total_incidents_timeout_escalated"`
	TotalIncidentsReassigned        int     `json:"total_incidents_reassigned" toon:"total_incidents_reassigned"`
	TotalInterruptions              int     `json:"total_interruptions" toon:"total_interruptions"`
	TotalNotifications              int     `json:"total_notifications" toon:"total_notifications"`
	TotalEngagedSeconds             int     `json:"total_engaged_seconds" toon:"total_engaged_seconds"`
	TotalSecondsToAck               int     `json:"total_seconds_to_ack" toon:"total_seconds_to_ack"`
	TotalSecondsToClose             int     `json:"total_seconds_to_close" toon:"total_seconds_to_close"`
	MeanSecondsToAck                float64 `json:"mean_seconds_to_ack" toon:"mean_seconds_to_ack"`
	MeanSecondsToClose              float64 `json:"mean_seconds_to_close" toon:"mean_seconds_to_close"`
	NoiseReductionPct               float64 `json:"noise_reduction_pct" toon:"noise_reduction_pct"`
	AcknowledgementPct              float64 `json:"acknowledgement_pct" toon:"acknowledgement_pct"`
	TotalAlertCnt                   int     `json:"total_alert_cnt" toon:"total_alert_cnt"`
	TotalAlertEventCnt              int     `json:"total_alert_event_cnt" toon:"total_alert_event_cnt"`
}

// ResponderInsightItem represents per-responder pre-aggregated metrics.
type ResponderInsightItem struct {
	MetricsBase

	Email                           string  `json:"email,omitempty" toon:"email,omitempty"`
	TotalIncidentCnt                int     `json:"total_incident_cnt" toon:"total_incident_cnt"`
	TotalIncidentsAcknowledged      int     `json:"total_incidents_acknowledged" toon:"total_incidents_acknowledged"`
	TotalIncidentsReassigned        int     `json:"total_incidents_reassigned" toon:"total_incidents_reassigned"`
	TotalIncidentsEscalated         int     `json:"total_incidents_escalated" toon:"total_incidents_escalated"`
	TotalIncidentsTimeoutEscalated  int     `json:"total_incidents_timeout_escalated" toon:"total_incidents_timeout_escalated"`
	TotalIncidentsManuallyEscalated int     `json:"total_incidents_manually_escalated" toon:"total_incidents_manually_escalated"`
	TotalInterruptions              int     `json:"total_interruptions" toon:"total_interruptions"`
	TotalNotifications              int     `json:"total_notifications" toon:"total_notifications"`
	TotalEngagedSeconds             int     `json:"total_engaged_seconds" toon:"total_engaged_seconds"`
	TotalSecondsToAck               int     `json:"total_seconds_to_ack" toon:"total_seconds_to_ack"`
	MeanSecondsToAck                float64 `json:"mean_seconds_to_ack" toon:"mean_seconds_to_ack"`
	AcknowledgementPct              float64 `json:"acknowledgement_pct" toon:"acknowledgement_pct"`
}

// InsightAlertByLabelItem represents a top-K alert source grouped by label
type InsightAlertByLabelItem struct {
	Label              string `json:"label" toon:"label"`
	Hours              string `json:"hours,omitempty" toon:"hours,omitempty"`
	TotalAlertCnt      int    `json:"total_alert_cnt" toon:"total_alert_cnt"`
	TotalAlertEventCnt int    `json:"total_alert_event_cnt" toon:"total_alert_event_cnt"`
}

// InsightIncidentItem represents an incident with attached performance metrics from the insight API
type InsightIncidentItem struct {
	IncidentID         string            `json:"incident_id" toon:"incident_id"`
	Title              string            `json:"title" toon:"title"`
	Description        string            `json:"description,omitempty" toon:"description,omitempty"`
	TeamID             int64             `json:"team_id,omitempty" toon:"team_id,omitempty"`
	TeamName           string            `json:"team_name,omitempty" toon:"team_name,omitempty"`
	ChannelID          int64             `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName        string            `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	Progress           string            `json:"progress" toon:"progress"`
	Severity           string            `json:"severity" toon:"severity"`
	CreatedAt          int64             `json:"created_at" toon:"created_at"`
	ClosedBy           string            `json:"closed_by,omitempty" toon:"closed_by,omitempty"`
	SecondsToAck       int               `json:"seconds_to_ack" toon:"seconds_to_ack"`
	SecondsToClose     int               `json:"seconds_to_close" toon:"seconds_to_close"`
	EngagedSeconds     int               `json:"engaged_seconds" toon:"engaged_seconds"`
	Hours              string            `json:"hours,omitempty" toon:"hours,omitempty"`
	Responders         []RawResponder    `json:"responders,omitempty" toon:"responders,omitempty"`
	AssignedTo         *AssignedTo       `json:"assigned_to,omitempty" toon:"assigned_to,omitempty"`
	Labels             map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
	Fields             map[string]any    `json:"fields,omitempty" toon:"fields,omitempty"`
	Notifications      int               `json:"notifications,omitempty" toon:"notifications,omitempty"`
	Interruptions      int               `json:"interruptions,omitempty" toon:"interruptions,omitempty"`
	Assignments        int               `json:"assignments,omitempty" toon:"assignments,omitempty"`
	Reassignments      int               `json:"reassignments,omitempty" toon:"reassignments,omitempty"`
	Acknowledgements   int               `json:"acknowledgements,omitempty" toon:"acknowledgements,omitempty"`
	Escalations        int               `json:"escalations,omitempty" toon:"escalations,omitempty"`
	TimeoutEscalations int               `json:"timeout_escalations,omitempty" toon:"timeout_escalations,omitempty"`
	ManualEscalations  int               `json:"manual_escalations,omitempty" toon:"manual_escalations,omitempty"`
	CreatorID          int64             `json:"creator_id,omitempty" toon:"creator_id,omitempty"`
	CreatorName        string            `json:"creator_name,omitempty" toon:"creator_name,omitempty"`
}

// IncidentDetail represents full incident data from the /incident/info endpoint
type IncidentDetail struct {
	IncidentID    string            `json:"incident_id" toon:"incident_id"`
	Title         string            `json:"title" toon:"title"`
	Description   string            `json:"description,omitempty" toon:"description,omitempty"`
	Severity      string            `json:"incident_severity" toon:"severity"`
	Progress      string            `json:"progress" toon:"progress"`
	StartTime     int64             `json:"start_time" toon:"start_time"`
	AckTime       int64             `json:"ack_time,omitempty" toon:"ack_time,omitempty"`
	CloseTime     int64             `json:"close_time,omitempty" toon:"close_time,omitempty"`
	ChannelID     int64             `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName   string            `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	CreatorID     int64             `json:"creator_id,omitempty" toon:"creator_id,omitempty"`
	CloserID      int64             `json:"closer_id,omitempty" toon:"closer_id,omitempty"`
	AISummary     string            `json:"ai_summary,omitempty" toon:"ai_summary,omitempty"`
	RootCause     string            `json:"root_cause,omitempty" toon:"root_cause,omitempty"`
	Resolution    string            `json:"resolution,omitempty" toon:"resolution,omitempty"`
	Impact        string            `json:"impact,omitempty" toon:"impact,omitempty"`
	Frequency     string            `json:"frequency,omitempty" toon:"frequency,omitempty"`
	AlertCnt      int               `json:"alert_cnt" toon:"alert_cnt"`
	AlertEventCnt int               `json:"alert_event_cnt" toon:"alert_event_cnt"`
	PostMortemID  string            `json:"post_mortem_id,omitempty" toon:"post_mortem_id,omitempty"`
	Responders    []RawResponder    `json:"responders,omitempty" toon:"responders,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
	Fields        map[string]any    `json:"fields,omitempty" toon:"fields,omitempty"`
}

// PostMortem represents post-mortem metadata returned by /incident/post-mortem/list.
type PostMortem struct {
	AccountID        int64    `json:"account_id,omitempty" toon:"account_id,omitempty"`
	PostMortemID     string   `json:"post_mortem_id" toon:"post_mortem_id"`
	TemplateID       string   `json:"template_id,omitempty" toon:"template_id,omitempty"`
	IncidentIDs      []string `json:"incident_ids,omitempty" toon:"incident_ids,omitempty"`
	MediaCount       int      `json:"media_count,omitempty" toon:"media_count,omitempty"`
	AuthorIDs        []int64  `json:"author_ids,omitempty" toon:"author_ids,omitempty"`
	TeamID           int64    `json:"team_id,omitempty" toon:"team_id,omitempty"`
	ChannelID        int64    `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName      string   `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	IsPrivate        bool     `json:"is_private,omitempty" toon:"is_private,omitempty"`
	Title            string   `json:"title,omitempty" toon:"title,omitempty"`
	Status           string   `json:"status,omitempty" toon:"status,omitempty"`
	CreatedAtSeconds int64    `json:"created_at_seconds,omitempty" toon:"created_at_seconds,omitempty"`
	UpdatedAtSeconds int64    `json:"updated_at_seconds,omitempty" toon:"updated_at_seconds,omitempty"`
}

// AlertEvent represents a raw alert event
type AlertEvent struct {
	EventID         string            `json:"event_id" toon:"event_id"`
	AlertID         string            `json:"alert_id,omitempty" toon:"alert_id,omitempty"`
	AccountID       int64             `json:"account_id,omitempty" toon:"account_id,omitempty"`
	ChannelID       int64             `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	IntegrationID   int64             `json:"integration_id,omitempty" toon:"integration_id,omitempty"`
	IntegrationType string            `json:"integration_type,omitempty" toon:"integration_type,omitempty"`
	Title           string            `json:"title,omitempty" toon:"title,omitempty"`
	Description     string            `json:"description,omitempty" toon:"description,omitempty"`
	EventSeverity   string            `json:"event_severity" toon:"event_severity"`
	EventStatus     string            `json:"event_status" toon:"event_status"`
	EventTime       int64             `json:"event_time" toon:"event_time"`
	Labels          map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
	CreatedAt       int64             `json:"created_at,omitempty" toon:"created_at,omitempty"`
	UpdatedAt       int64             `json:"updated_at,omitempty" toon:"updated_at,omitempty"`
}

// Alert represents a deduplicated alert entity (Layer 1)
type Alert struct {
	AlertID         string            `json:"alert_id" toon:"alert_id"`
	ChannelID       int64             `json:"channel_id,omitempty" toon:"channel_id,omitempty"`
	ChannelName     string            `json:"channel_name,omitempty" toon:"channel_name,omitempty"`
	IntegrationID   int64             `json:"integration_id,omitempty" toon:"integration_id,omitempty"`
	IntegrationName string            `json:"integration_name,omitempty" toon:"integration_name,omitempty"`
	IntegrationType string            `json:"integration_type,omitempty" toon:"integration_type,omitempty"`
	Title           string            `json:"title" toon:"title"`
	Description     string            `json:"description,omitempty" toon:"description,omitempty"`
	AlertKey        string            `json:"alert_key,omitempty" toon:"alert_key,omitempty"`
	AlertSeverity   string            `json:"alert_severity" toon:"alert_severity"`
	AlertStatus     string            `json:"alert_status" toon:"alert_status"`
	StartTime       int64             `json:"start_time" toon:"start_time"`
	LastTime        int64             `json:"last_time,omitempty" toon:"last_time,omitempty"`
	EndTime         int64             `json:"end_time,omitempty" toon:"end_time,omitempty"`
	EventCnt        int               `json:"event_cnt,omitempty" toon:"event_cnt,omitempty"`
	EverMuted       bool              `json:"ever_muted,omitempty" toon:"ever_muted,omitempty"`
	Labels          map[string]string `json:"labels,omitempty" toon:"labels,omitempty"`
	Incident        *AlertIncident    `json:"incident,omitempty" toon:"incident,omitempty"`
}

// AlertIncident is the parent incident reference embedded in an alert
type AlertIncident struct {
	IncidentID string `json:"incident_id" toon:"incident_id"`
	Title      string `json:"title,omitempty" toon:"title,omitempty"`
	Progress   string `json:"progress,omitempty" toon:"progress,omitempty"`
}
