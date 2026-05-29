package flashduty

import (
	"context"
	"fmt"
)

// IncidentNotifyInput contains optional notification controls for incident write operations.
type IncidentNotifyInput struct {
	FollowPreference bool
	PersonalChannels []string
	TemplateID       string
}

// IncidentCommentInput contains parameters for adding a comment to one or more incidents.
type IncidentCommentInput struct {
	IncidentIDs []string
	Comment     string
	MuteReply   bool
	Notify      *IncidentNotifyInput
}

// IncidentAddResponderInput contains parameters for adding responders to an incident.
type IncidentAddResponderInput struct {
	IncidentID string
	PersonIDs  []int64
	Notify     *IncidentNotifyInput
}

// IncidentWarRoomCreateInput contains parameters for creating an incident war room.
type IncidentWarRoomCreateInput struct {
	IncidentID    string
	IntegrationID int64
	MemberIDs     []int64
	AddObservers  bool
}

// IncidentWarRoomListInput contains parameters for listing war rooms on an incident.
type IncidentWarRoomListInput struct {
	IncidentID    string
	IntegrationID int64
}

// IncidentWarRoomDetailInput contains parameters for fetching a war room detail.
type IncidentWarRoomDetailInput struct {
	IntegrationID int64
	ChatID        string
}

// IncidentWarRoomDeleteInput contains parameters for deleting an incident war room.
type IncidentWarRoomDeleteInput struct {
	IncidentID    string
	IntegrationID int64
}

// IncidentWarRoomAddMemberInput contains parameters for adding members to a war room.
type IncidentWarRoomAddMemberInput struct {
	IntegrationID int64
	ChatID        string
	MemberIDs     []int64
}

// IncidentWarRoom represents an IM war room.
type IncidentWarRoom struct {
	AccountID     int64     `json:"account_id,omitempty" toon:"account_id,omitempty"`
	IntegrationID int64     `json:"integration_id,omitempty" toon:"integration_id,omitempty"`
	ChatID        string    `json:"chat_id" toon:"chat_id"`
	ChatName      string    `json:"chat_name,omitempty" toon:"chat_name,omitempty"`
	ShareLink     string    `json:"share_link,omitempty" toon:"share_link,omitempty"`
	IncidentID    string    `json:"incident_id,omitempty" toon:"incident_id,omitempty"`
	CreatedBy     int64     `json:"created_by,omitempty" toon:"created_by,omitempty"`
	CreatedAt     Timestamp `json:"created_at,omitempty" toon:"created_at,omitempty"`
	PluginType    string    `json:"plugin_type,omitempty" toon:"plugin_type,omitempty"`
	Status        string    `json:"status,omitempty" toon:"status,omitempty"`
}

// IncidentWarRoomItem represents a war room list item.
type IncidentWarRoomItem = IncidentWarRoom

// IncidentWarRoomListOutput contains war rooms for an incident.
type IncidentWarRoomListOutput struct {
	Items []IncidentWarRoomItem `json:"items" toon:"items"`
}

// IncidentWarRoomObserver represents a default observer candidate for war room invitation.
type IncidentWarRoomObserver struct {
	PersonID   int64     `json:"person_id" toon:"person_id"`
	PersonName string    `json:"person_name,omitempty" toon:"person_name,omitempty"`
	Nickname   string    `json:"nickname,omitempty" toon:"nickname,omitempty"`
	Name       string    `json:"name,omitempty" toon:"name,omitempty"`
	Email      string    `json:"email,omitempty" toon:"email,omitempty"`
	Phone      string    `json:"phone,omitempty" toon:"phone,omitempty"`
	Status     string    `json:"status,omitempty" toon:"status,omitempty"`
	AssignedAt Timestamp `json:"assigned_at,omitempty" toon:"assigned_at,omitempty"`
}

// DisplayName returns the best available human-readable observer name.
func (o IncidentWarRoomObserver) DisplayName() string {
	if o.PersonName != "" {
		return o.PersonName
	}
	if o.Nickname != "" {
		return o.Nickname
	}
	return o.Name
}

// UnackIncidents cancels acknowledgement for one or more incidents.
func (c *Client) UnackIncidents(ctx context.Context, incidentIDs []string) error {
	return postEmpty(c, ctx, "/incident/unack", map[string]any{"incident_ids": incidentIDs}, "failed to unack incidents")
}

// WakeIncidents wakes one or more incidents from snooze.
func (c *Client) WakeIncidents(ctx context.Context, incidentIDs []string) error {
	return postEmpty(c, ctx, "/incident/wake", map[string]any{"incident_ids": incidentIDs}, "failed to wake incidents")
}

// RemoveIncidents removes one or more incidents.
func (c *Client) RemoveIncidents(ctx context.Context, incidentIDs []string) error {
	return postEmpty(c, ctx, "/incident/remove", map[string]any{"incident_ids": incidentIDs}, "failed to remove incidents")
}

// DisableIncidentMerge disables merge for one or more incidents.
func (c *Client) DisableIncidentMerge(ctx context.Context, incidentIDs []string) error {
	return postEmpty(c, ctx, "/incident/disable-merge", map[string]any{"incident_ids": incidentIDs}, "failed to disable incident merge")
}

// CommentIncidents adds a comment to one or more incidents.
func (c *Client) CommentIncidents(ctx context.Context, input *IncidentCommentInput) error {
	if input == nil {
		return fmt.Errorf("incident comment input is required")
	}
	body := map[string]any{
		"incident_ids": input.IncidentIDs,
		"comment":      input.Comment,
	}
	if input.MuteReply {
		body["mute_reply"] = true
	}
	if notify := buildIncidentNotifyBody(input.Notify); len(notify) > 0 {
		body["notify"] = notify
	}
	return postEmpty(c, ctx, "/incident/comment", body, "failed to comment incidents")
}

// AddIncidentResponders adds responders to an incident.
func (c *Client) AddIncidentResponders(ctx context.Context, input *IncidentAddResponderInput) error {
	if input == nil {
		return fmt.Errorf("incident add responder input is required")
	}
	body := map[string]any{
		"incident_id": input.IncidentID,
		"person_ids":  input.PersonIDs,
	}
	if notify := buildIncidentNotifyBody(input.Notify); len(notify) > 0 {
		body["notify"] = notify
	}
	return postEmpty(c, ctx, "/incident/responder/add", body, "failed to add incident responders")
}

// CreateIncidentWarRoom creates an IM war room for an incident.
func (c *Client) CreateIncidentWarRoom(ctx context.Context, input *IncidentWarRoomCreateInput) (*IncidentWarRoom, error) {
	if input == nil {
		return nil, fmt.Errorf("incident war-room create input is required")
	}
	body := map[string]any{
		"incident_id":    input.IncidentID,
		"integration_id": input.IntegrationID,
	}
	if len(input.MemberIDs) > 0 {
		body["member_ids"] = input.MemberIDs
	}
	if input.AddObservers {
		body["add_observers"] = true
	}
	return postData[IncidentWarRoom](c, ctx, "/incident/war-room/create", body, "failed to create incident war room")
}

// ListIncidentWarRooms lists war rooms for an incident.
func (c *Client) ListIncidentWarRooms(ctx context.Context, input *IncidentWarRoomListInput) (*IncidentWarRoomListOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("incident war-room list input is required")
	}
	body := map[string]any{"incident_id": input.IncidentID}
	if input.IntegrationID > 0 {
		body["integration_id"] = input.IntegrationID
	}
	return postData[IncidentWarRoomListOutput](c, ctx, "/incident/war-room/list", body, "failed to list incident war rooms")
}

// GetIncidentWarRoom fetches war room details from the IM provider.
func (c *Client) GetIncidentWarRoom(ctx context.Context, input *IncidentWarRoomDetailInput) (*IncidentWarRoom, error) {
	if input == nil {
		return nil, fmt.Errorf("incident war-room detail input is required")
	}
	return postData[IncidentWarRoom](c, ctx, "/incident/war-room/detail", map[string]any{
		"integration_id": input.IntegrationID,
		"chat_id":        input.ChatID,
	}, "failed to get incident war room")
}

// DeleteIncidentWarRoom deletes an incident war room.
func (c *Client) DeleteIncidentWarRoom(ctx context.Context, input *IncidentWarRoomDeleteInput) error {
	if input == nil {
		return fmt.Errorf("incident war-room delete input is required")
	}
	return postEmpty(c, ctx, "/incident/war-room/delete", map[string]any{
		"incident_id":    input.IncidentID,
		"integration_id": input.IntegrationID,
	}, "failed to delete incident war room")
}

// AddIncidentWarRoomMembers adds members to an existing incident war room.
func (c *Client) AddIncidentWarRoomMembers(ctx context.Context, input *IncidentWarRoomAddMemberInput) error {
	if input == nil {
		return fmt.Errorf("incident war-room add-member input is required")
	}
	return postEmpty(c, ctx, "/incident/war-room/add-member", map[string]any{
		"integration_id": input.IntegrationID,
		"chat_id":        input.ChatID,
		"member_ids":     input.MemberIDs,
	}, "failed to add incident war room members")
}

// GetIncidentWarRoomDefaultObservers returns historical responders eligible for observer invitation.
func (c *Client) GetIncidentWarRoomDefaultObservers(ctx context.Context, incidentID string) ([]IncidentWarRoomObserver, error) {
	out, err := postData[struct {
		Observers []IncidentWarRoomObserver `json:"observers"`
	}](c, ctx, "/incident/war-room/default-observers", map[string]any{
		"incident_id": incidentID,
	}, "failed to get incident war room default observers")
	if err != nil {
		return nil, err
	}
	return out.Observers, nil
}

func buildIncidentNotifyBody(input *IncidentNotifyInput) map[string]any {
	notify := map[string]any{}
	if input == nil {
		return notify
	}
	if input.FollowPreference {
		notify["follow_preference"] = true
	}
	if len(input.PersonalChannels) > 0 {
		notify["personal_channels"] = input.PersonalChannels
	}
	if input.TemplateID != "" {
		notify["template_id"] = input.TemplateID
	}
	return notify
}
