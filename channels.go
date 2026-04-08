package flashduty

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ListChannelsInput contains parameters for listing channels
type ListChannelsInput struct {
	ChannelIDs []int64 // Direct lookup by channel IDs
	Name       string  // Search by channel name (case-insensitive substring match, done client-side)
}

// ListChannelsOutput contains the result of listing channels
type ListChannelsOutput struct {
	Channels []ChannelInfo `json:"channels"`
	Total    int           `json:"total"`
}

// ListChannels queries channels by IDs or name, returns enriched data with team/creator names
func (c *Client) ListChannels(ctx context.Context, input *ListChannelsInput) (*ListChannelsOutput, error) {
	// Query by channel IDs
	if len(input.ChannelIDs) > 0 {
		channelMap, err := c.fetchChannelInfos(ctx, input.ChannelIDs)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve channels: %w", err)
		}

		channels := make([]ChannelInfo, 0, len(channelMap))
		for _, ch := range channelMap {
			channels = append(channels, ch)
		}

		enrichedChannels, err := c.enrichChannels(ctx, channels)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch team and creator details: %w", err)
		}

		return &ListChannelsOutput{
			Channels: enrichedChannels,
			Total:    len(enrichedChannels),
		}, nil
	}

	// List all channels
	resp, err := c.makeRequest(ctx, "POST", "/channel/list", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("unable to list channels: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []struct {
				ChannelID   int64  `json:"channel_id"`
				ChannelName string `json:"channel_name"`
				TeamID      int64  `json:"team_id,omitempty"`
				CreatorID   int64  `json:"creator_id,omitempty"`
			} `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	channels := []ChannelInfo{}
	if result.Data != nil {
		for _, ch := range result.Data.Items {
			// Filter by name if provided (case-insensitive substring match)
			if input.Name != "" && !strings.Contains(strings.ToLower(ch.ChannelName), strings.ToLower(input.Name)) {
				continue
			}
			channels = append(channels, ChannelInfo{
				ChannelID:   ch.ChannelID,
				ChannelName: ch.ChannelName,
				TeamID:      ch.TeamID,
				CreatorID:   ch.CreatorID,
			})
		}
	}

	enrichedChannels, err := c.enrichChannels(ctx, channels)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team and creator details: %w", err)
	}

	return &ListChannelsOutput{
		Channels: enrichedChannels,
		Total:    len(enrichedChannels),
	}, nil
}
