package flashduty

import (
	"context"
	"fmt"
	"slices"
)

// PresetTemplateID is the fixed MongoDB ObjectID for the system preset template
const PresetTemplateID = "6321aad26c12104586a88916"

// TemplateChannels maps channel identifiers to TemplateDoc field names.
var TemplateChannels = map[string]string{
	"dingtalk":     "dingtalk",
	"dingtalk_app": "dingtalk_app",
	"feishu":       "feishu",
	"feishu_app":   "feishu_app",
	"wecom":        "wecom",
	"wecom_app":    "wecom_app",
	"slack":        "slack",
	"slack_app":    "slack_app",
	"telegram":     "telegram",
	"teams_app":    "teams_app",
	"email":        "email",
	"sms":          "sms",
	"zoom":         "zoom",
}

// ChannelSizeLimits defines the maximum rendered size per channel.
// 0 means no enforced limit.
var ChannelSizeLimits = map[string]int{
	"dingtalk":     4000,
	"dingtalk_app": 0,
	"feishu":       4000,
	"feishu_app":   0,
	"wecom":        4000,
	"wecom_app":    0,
	"slack":        15000,
	"slack_app":    15000,
	"telegram":     4096,
	"teams_app":    28000,
	"email":        0,
	"sms":          0,
	"zoom":         0,
}

// ChannelEnumValues returns all valid notification channel identifiers.
func ChannelEnumValues() []string {
	channels := make([]string, 0, len(TemplateChannels))
	for k := range TemplateChannels {
		channels = append(channels, k)
	}
	slices.Sort(channels)
	return channels
}

// GetPresetTemplateInput contains parameters for fetching a preset template.
type GetPresetTemplateInput struct {
	Channel string // Required. One of the keys in TemplateChannels.
}

// GetPresetTemplateOutput contains the preset template result.
type GetPresetTemplateOutput struct {
	Channel      string `json:"channel"`
	FieldName    string `json:"field_name"`
	TemplateCode string `json:"template_code"`
}

// GetPresetTemplate fetches the preset (default) notification template for a channel.
func (c *Client) GetPresetTemplate(ctx context.Context, input *GetPresetTemplateInput) (*GetPresetTemplateOutput, error) {
	fieldName, ok := TemplateChannels[input.Channel]
	if !ok {
		return nil, fmt.Errorf("unknown channel: %s", input.Channel)
	}

	requestBody := map[string]any{
		"id": PresetTemplateID,
	}

	result, err := postData[map[string]interface{}](c, ctx, "/template/info", requestBody, "failed to fetch preset template")
	if err != nil {
		return nil, err
	}

	templateCode := ""
	if result != nil {
		if val, ok := (*result)[fieldName]; ok {
			if str, ok := val.(string); ok {
				templateCode = str
			}
		}
	}

	if templateCode == "" {
		return nil, fmt.Errorf("no preset template found for channel: %s", input.Channel)
	}

	return &GetPresetTemplateOutput{
		Channel:      input.Channel,
		FieldName:    fieldName,
		TemplateCode: templateCode,
	}, nil
}

// ValidateTemplateInput contains parameters for validating a template.
type ValidateTemplateInput struct {
	Channel      string // Required. One of the keys in TemplateChannels.
	TemplateCode string // Required. The Go template code to validate.
	IncidentID   string // Optional. Real incident ID for preview; uses mock data if empty.
}

// ValidateTemplateOutput contains the validation result.
type ValidateTemplateOutput struct {
	Channel         string   `json:"channel"`
	FieldName       string   `json:"field_name"`
	TemplateCode    string   `json:"template_code"`
	Success         bool     `json:"success"`
	RenderedPreview string   `json:"rendered_preview"`
	RenderedSize    int      `json:"rendered_size"`
	SizeLimit       int      `json:"size_limit"`
	Errors          []string `json:"errors"`
	Warnings        []string `json:"warnings"`
}

// ValidateTemplate validates and previews a notification template.
func (c *Client) ValidateTemplate(ctx context.Context, input *ValidateTemplateInput) (*ValidateTemplateOutput, error) {
	fieldName, ok := TemplateChannels[input.Channel]
	if !ok {
		return nil, fmt.Errorf("unknown channel: %s", input.Channel)
	}

	requestBody := map[string]any{
		"content": input.TemplateCode,
		"type":    input.Channel,
	}
	if input.IncidentID != "" {
		requestBody["incident_id"] = input.IncidentID
	}

	result, err := postOptionalData[struct {
		Success bool   `json:"success"`
		Content string `json:"content"`
		Message string `json:"message"`
	}](c, ctx, "/template/preview", requestBody, "failed to validate template")
	if err != nil {
		return nil, err
	}

	success := false
	renderedPreview := ""
	errorMessage := ""
	if result != nil {
		success = result.Success
		renderedPreview = result.Content
		errorMessage = result.Message
	}

	renderedSize := len(renderedPreview)
	sizeLimit := ChannelSizeLimits[input.Channel]

	errs := []string{}
	warnings := []string{}

	if !success {
		errs = append(errs, errorMessage)
	}

	if sizeLimit > 0 {
		if renderedSize > sizeLimit {
			sizeWarning := fmt.Sprintf("Rendered output is %d bytes, exceeding the %d byte limit for %s.", renderedSize, sizeLimit, input.Channel)
			switch input.Channel {
			case "telegram":
				sizeWarning += " CRITICAL: Telegram will silently drop this message."
			case "teams_app":
				sizeWarning += " Teams will return an error for this message."
			}
			errs = append(errs, sizeWarning)
		} else if renderedSize > int(float64(sizeLimit)*0.8) {
			warnings = append(warnings, fmt.Sprintf("Rendered output is %d/%d bytes (%.0f%% of limit).", renderedSize, sizeLimit, float64(renderedSize)/float64(sizeLimit)*100))
		}
	}

	return &ValidateTemplateOutput{
		Channel:         input.Channel,
		FieldName:       fieldName,
		TemplateCode:    input.TemplateCode,
		Success:         success && len(errs) == 0,
		RenderedPreview: renderedPreview,
		RenderedSize:    renderedSize,
		SizeLimit:       sizeLimit,
		Errors:          errs,
		Warnings:        warnings,
	}, nil
}

// TemplateVariables returns the list of available template variables.
//
// NOTE: This data is compiled into the SDK. Platform-side additions require an SDK release.
func TemplateVariables() []TemplateVariable {
	result := make([]TemplateVariable, len(templateVariables))
	copy(result, templateVariables)
	return result
}

// TemplateCustomFunctions returns the list of custom Flashduty template functions.
//
// NOTE: This data is compiled into the SDK. Platform-side additions require an SDK release.
func TemplateCustomFunctions() []TemplateFunction {
	result := make([]TemplateFunction, len(templateCustomFunctions))
	copy(result, templateCustomFunctions)
	return result
}

// TemplateSprigFunctions returns the list of commonly used Sprig template functions.
//
// NOTE: This data is compiled into the SDK. Platform-side additions require an SDK release.
func TemplateSprigFunctions() []TemplateFunction {
	result := make([]TemplateFunction, len(templateSprigFunctions))
	copy(result, templateSprigFunctions)
	return result
}

// --- Static Data ---

var templateVariables = []TemplateVariable{
	// Core fields
	{".Title", "string", "Incident title", "Order Message Failed", "core"},
	{".Description", "string", "Incident description", "Send order message failed too many times", "core"},
	{".Num", "string", "Short incident number", "ABC123", "core"},
	{".ID", "string", "Incident ID", "6321aad26c12104586a88916", "core"},
	{".IncidentSeverity", "string", "Severity level: Critical, Warning, Info, Ok", "Critical", "core"},
	{".IncidentStatus", "string", "Status code: Critical, Warning, Info, Ok", "Critical", "core"},
	{".Progress", "string", "Handling progress: Triggered, Processing, Closed", "Triggered", "core"},
	{".DetailUrl", "string", "Link to incident detail page", "https://console.flashcat.com/incident/detail/...", "core"},

	// Time fields
	{".StartTime", "int64", "Unix timestamp - incident start", "", "time"},
	{".LastTime", "int64", "Unix timestamp - last update", "", "time"},
	{".AckTime", "int64", "Unix timestamp - acknowledgement (0 if not acked)", "", "time"},
	{".CloseTime", "int64", "Unix timestamp - closure (0 if not closed)", "", "time"},
	{".SnoozedBefore", "int64", "Unix timestamp - snooze expiry", "", "time"},

	// People fields
	{".Creator", "*PersonItem", "Incident creator: {PersonID, PersonName, Email}", "", "people"},
	{".Closer", "*PersonItem", "Person who closed the incident", "", "people"},
	{".Owner", "*PersonItem", "Current incident owner", "", "people"},
	{".Responders", "[]*Responder", "List of responders: {PersonID, PersonName, Email, AssignedAt, AcknowledgedAt}", "", "people"},
	{".AssignedTo", "*AssignedTo", "Assignment info: {EscalateRuleID, EscalateRuleName, LayerIdx, Type}", "", "people"},

	// Alert aggregation
	{".AlertCnt", "int64", "Total associated alerts count", "10", "alerts"},
	{".ActiveAlertCnt", "int64", "Active (non-resolved) alerts count", "9", "alerts"},
	{".AlertEventCnt", "int64", "Total alert events count", "30", "alerts"},
	{".Alerts", "[]*AlertItem", "Alert list: {Title, Description, AlertSeverity, AlertStatus, StartTime, LastTime, EndTime, Labels}", "", "alerts"},

	// Labels and custom data
	{".Labels", "map[string]string", "Alert label key-value pairs. Access via .Labels.key or index .Labels \"dotted.key\"", "", "labels"},
	{".Fields", "map[string]interface{}", "Custom incident fields", "", "labels"},
	{".Images", "[]Image", "Associated images: {Src, Alt}", "", "labels"},

	// Context fields
	{".ChannelName", "string", "Collaboration space name", "Order system", "context"},
	{".ChannelID", "int64", "Collaboration space ID", "", "context"},
	{".AccountName", "string", "Account/organization name", "Flashduty", "context"},
	{".AccountLocale", "string", "Locale: zh-CN or en-US", "zh-CN", "context"},
	{".AccountTimeZone", "string", "Account timezone", "", "context"},

	// Notification fields
	{".FireType", "string", "Notification type: fire (initial) or refire (recurring)", "fire", "notification"},
	{".FireTimes", "int64", "Number of times notified", "", "notification"},
	{".IsFlapping", "bool", "Whether in flapping state", "true", "notification"},
	{".IsInStorm", "bool", "Whether in alert storm", "false", "notification"},
	{".Flapping", "*Flapping", "Flapping config: {MaxChanges, InMinutes, MuteMinutes}", "", "notification"},
	{".GroupMethod", "string", "Grouping method: n (none), p (by rule), i (intelligent)", "i", "notification"},

	// Post-incident fields
	{".Impact", "string", "Impact description", "", "post_incident"},
	{".RootCause", "string", "Root cause", "", "post_incident"},
	{".Resolution", "string", "Resolution description", "", "post_incident"},
	{".AISummary", "string", "AI-generated incident summary", "", "post_incident"},
}

var templateCustomFunctions = []TemplateFunction{
	{"date", `{{date "2006-01-02 15:04:05" .StartTime}}`, "Format Unix timestamp using Go time layout"},
	{"ago", `{{ago .StartTime}}`, "Human-readable duration since timestamp (e.g., '2 hours ago')"},
	{"toHtml", `{{toHtml .Title}}`, "HTML-escape special characters; accepts multiple args, uses first non-empty"},
	{"fireReason", `{{fireReason .}}`, "Returns notification type prefix: [REFIRE], [ESCALATE], etc."},
	{"colorSeverity", `{{colorSeverity .IncidentSeverity}}`, "Severity with <font color> markup for chat platforms"},
	{"colorBySeverity", `{{colorBySeverity .IncidentSeverity "text"}}`, "Color any text using severity-based color"},
	{"serverityToColor", `{{serverityToColor .IncidentSeverity}}`, "Returns hex color: #C80000 (Critical), #FA7D00 (Warning), #FABE00 (Info), #008800 (Ok)"},
	{"toSeverity", `{{toSeverity .IncidentSeverity}}`, "Severity to localized display string"},
	{"joinAlertLabels", `{{joinAlertLabels . "resource" ", "}}`, "Deduplicate and join a label's values from all alerts"},
	{"alertLabels", `{{alertLabels . "resource"}}`, "Return deduplicated label values as array"},
	{"maxAlertLabel", `{{maxAlertLabel . "trigger_value"}}`, "Max value of a label across alerts"},
	{"minAlertLabel", `{{minAlertLabel . "trigger_value"}}`, "Min value of a label across alerts"},
	{"in", `{{in $k "resource" "body_text"}}`, "Check if value is in a set of values"},
	{"mdToHtml", `{{mdToHtml .Description}}`, "Convert Markdown to sanitized HTML"},
	{"transferImage", `{{transferImage $root $v.Src}}`, "Upload image to Feishu (Feishu App only)"},
	{"imageSrcToURL", `{{imageSrcToURL $root $v.Src}}`, "Convert image key to accessible URL (DingTalk, Slack)"},
	{"imageAltToURL", `{{imageAltToURL $root $v.Alt}}`, "Get image URL by alt text"},
	{"jsonGet", `{{jsonGet .Labels.rule_note "detail_url"}}`, "Parse JSON string and extract via gjson path syntax"},
	{"index", `{{index .Labels "dotted.key"}}`, "Access map keys containing dots"},
}

var templateSprigFunctions = []TemplateFunction{
	{"trim", `{{trim .Title}}`, "Remove leading/trailing whitespace"},
	{"upper", `{{upper .IncidentSeverity}}`, "Convert to uppercase"},
	{"lower", `{{lower .IncidentSeverity}}`, "Convert to lowercase"},
	{"replace", `{{replace "old" "new" .Title}}`, "Replace all occurrences"},
	{"contains", `{{contains "error" .Title}}`, "Check if string contains substring"},
	{"default", `{{default "N/A" .Description}}`, "Return default value if empty"},
	{"ternary", `{{ternary "yes" "no" .IsFlapping}}`, "Ternary operator"},
	{"add", `{{add .AlertCnt 1}}`, "Add numbers"},
	{"sub", `{{sub .AlertCnt 1}}`, "Subtract numbers"},
	{"len", `{{len .Responders}}`, "Length of list/map/string"},
	{"list", `{{list "a" "b" "c"}}`, "Create a list"},
	{"dict", `{{dict "key" "value"}}`, "Create a dictionary"},
	{"hasKey", `{{hasKey .Labels "resource"}}`, "Check if map has key"},
	{"keys", `{{keys .Labels}}`, "Get map keys"},
	{"values", `{{values .Labels}}`, "Get map values"},
	{"empty", `{{empty .Description}}`, "Check if value is empty/zero"},
	{"coalesce", `{{coalesce .Description "No description"}}`, "Return first non-empty value"},
	{"toString", `{{toString .AlertCnt}}`, "Convert to string"},
	{"toInt64", `{{toInt64 "123"}}`, "Convert to int64"},
}
