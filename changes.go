package flashduty

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
)

// ListChangesInput contains parameters for listing changes
type ListChangesInput struct {
	ChangeIDs []string // Direct lookup by change IDs
	ChannelID int64    // Filter by collaboration space ID
	StartTime int64    // Unix timestamp (seconds)
	EndTime   int64    // Unix timestamp (seconds)
	Type      string   // Filter by change type
	Limit     int      // Max results (default 20)
	Page      int      // Page number (default 1)
}

// ListChangesOutput contains the result of listing changes
type ListChangesOutput struct {
	Changes []Change `json:"changes"`
	Total   int      `json:"total"`
}

// ListChanges queries change records (deployments, configurations)
func (c *Client) ListChanges(ctx context.Context, input *ListChangesInput) (*ListChangesOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	requestBody := map[string]any{
		"p":     page,
		"limit": limit,
	}

	if len(input.ChangeIDs) > 0 {
		requestBody["change_ids"] = input.ChangeIDs
	}
	if input.ChannelID > 0 {
		requestBody["channel_id"] = input.ChannelID
	}
	if input.StartTime > 0 {
		requestBody["start_time"] = input.StartTime
	}
	if input.EndTime > 0 {
		requestBody["end_time"] = input.EndTime
	}
	if input.Type != "" {
		requestBody["type"] = input.Type
	}

	resp, err := c.makeRequest(ctx, "POST", "/change/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query changes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []struct {
				ChangeID    string            `json:"change_id"`
				Title       string            `json:"title"`
				Description string            `json:"description,omitempty"`
				Type        string            `json:"type,omitempty"`
				Status      string            `json:"status,omitempty"`
				ChannelID   int64             `json:"channel_id,omitempty"`
				CreatorID   int64             `json:"creator_id,omitempty"`
				StartTime   int64             `json:"start_time,omitempty"`
				EndTime     int64             `json:"end_time,omitempty"`
				Labels      map[string]string `json:"labels,omitempty"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	if result.Data == nil || len(result.Data.Items) == 0 {
		return &ListChangesOutput{
			Changes: []Change{},
			Total:   0,
		}, nil
	}

	// Collect IDs for enrichment
	channelIDs := make([]int64, 0)
	personIDs := make([]int64, 0)
	for _, item := range result.Data.Items {
		if item.ChannelID != 0 {
			channelIDs = append(channelIDs, item.ChannelID)
		}
		if item.CreatorID != 0 {
			personIDs = append(personIDs, item.CreatorID)
		}
	}

	// Fetch enrichment data concurrently (best-effort)
	var channelMap map[int64]ChannelInfo
	var personMap map[int64]PersonInfo
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		channelMap, _ = c.fetchChannelInfos(gctx, channelIDs)
		return nil
	})

	g.Go(func() error {
		personMap, _ = c.fetchPersonInfos(gctx, personIDs)
		return nil
	})

	_ = g.Wait()

	// Build enriched changes
	changes := make([]Change, 0, len(result.Data.Items))
	for _, item := range result.Data.Items {
		change := Change{
			ChangeID:    item.ChangeID,
			Title:       item.Title,
			Description: item.Description,
			Type:        item.Type,
			Status:      item.Status,
			ChannelID:   item.ChannelID,
			CreatorID:   item.CreatorID,
			StartTime:   item.StartTime,
			EndTime:     item.EndTime,
			Labels:      item.Labels,
		}

		if channelMap != nil {
			if ch, ok := channelMap[item.ChannelID]; ok {
				change.ChannelName = ch.ChannelName
			}
		}
		if personMap != nil {
			if p, ok := personMap[item.CreatorID]; ok {
				change.CreatorName = p.PersonName
			}
		}

		changes = append(changes, change)
	}

	return &ListChangesOutput{
		Changes: changes,
		Total:   result.Data.Total,
	}, nil
}
