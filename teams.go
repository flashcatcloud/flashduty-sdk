package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

const defaultTeamsQueryLimit = 20

// ListTeamsInput contains parameters for listing teams
type ListTeamsInput struct {
	TeamIDs  []int64 // Direct lookup by team IDs
	Name     string  // Search by team name
	Page     int     // Page number (default 1)
	Limit    int     // Page size (max 100, default 20)
	OrderBy  string  // Sort field: created_at, updated_at, team_name
	Asc      bool    // Ascending sort order
	PersonID int64   // Filter by member ID
}

// ListTeamsOutput contains the result of listing teams
type ListTeamsOutput struct {
	Teams []TeamInfo `json:"teams"`
	Total int        `json:"total"`
}

// ListTeams queries teams by IDs or name
func (c *Client) ListTeams(ctx context.Context, input *ListTeamsInput) (*ListTeamsOutput, error) {
	// Query by team IDs
	if len(input.TeamIDs) > 0 {
		requestBody := map[string]any{
			"team_ids": input.TeamIDs,
		}

		resp, err := c.makeRequest(ctx, "POST", "/team/infos", requestBody)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve teams: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return nil, handleAPIError(c.logger, resp)
		}

		var result struct {
			Error *DutyError `json:"error,omitempty"`
			Data  *struct {
				Items []struct {
					TeamID    int64   `json:"team_id"`
					TeamName  string  `json:"team_name"`
					PersonIDs []int64 `json:"person_ids"`
				} `json:"items"`
			} `json:"data,omitempty"`
		}
		if err := parseResponse(c.logger, resp, &result); err != nil {
			return nil, err
		}
		if result.Error != nil {
			return nil, result.Error
		}

		teams := []TeamInfo{}
		if result.Data != nil {
			for _, t := range result.Data.Items {
				teams = append(teams, TeamInfo{
					TeamID:    t.TeamID,
					TeamName:  t.TeamName,
					PersonIDs: t.PersonIDs,
				})
			}
		}

		c.enrichTeamMembers(ctx, teams)

		return &ListTeamsOutput{
			Teams: teams,
			Total: len(teams),
		}, nil
	}

	// List all teams
	page := input.Page
	if page <= 0 {
		page = 1
	}
	limit := input.Limit
	if limit <= 0 {
		limit = defaultTeamsQueryLimit
	}
	if limit > 100 {
		limit = 100
	}
	requestBody := map[string]any{
		"p":     page,
		"limit": limit,
	}
	if input.Name != "" {
		requestBody["query"] = input.Name
	}
	switch input.OrderBy {
	case "created_at", "updated_at", "team_name":
		requestBody["orderby"] = input.OrderBy
		requestBody["asc"] = input.Asc
	case "":
		// no ordering
	default:
		return nil, fmt.Errorf("invalid orderby value %q: must be one of created_at, updated_at, team_name", input.OrderBy)
	}
	if input.PersonID != 0 {
		requestBody["person_id"] = input.PersonID
	}

	resp, err := c.makeRequest(ctx, "POST", "/team/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to list teams: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []struct {
				TeamID    int64   `json:"team_id"`
				TeamName  string  `json:"team_name"`
				PersonIDs []int64 `json:"person_ids"`
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

	teams := []TeamInfo{}
	total := 0
	if result.Data != nil {
		for _, t := range result.Data.Items {
			teams = append(teams, TeamInfo{
				TeamID:    t.TeamID,
				TeamName:  t.TeamName,
				PersonIDs: t.PersonIDs,
			})
		}
		total = result.Data.Total
	}

	c.enrichTeamMembers(ctx, teams)

	return &ListTeamsOutput{
		Teams: teams,
		Total: total,
	}, nil
}

// enrichTeamMembers resolves member names for a slice of teams via /person/infos.
func (c *Client) enrichTeamMembers(ctx context.Context, teams []TeamInfo) {
	var allIDs []int64
	for _, t := range teams {
		allIDs = append(allIDs, t.PersonIDs...)
	}
	if len(allIDs) == 0 {
		return
	}

	personMap, err := c.fetchPersonInfos(ctx, allIDs)
	if err != nil {
		c.logger.Warn("failed to enrich team members", "error", err)
		return
	}

	for i := range teams {
		members := make([]TeamMember, 0, len(teams[i].PersonIDs))
		for _, pid := range teams[i].PersonIDs {
			if p, ok := personMap[pid]; ok {
				members = append(members, TeamMember{
					PersonID:   p.PersonID,
					PersonName: p.PersonName,
					Email:      p.Email,
				})
			} else {
				members = append(members, TeamMember{PersonID: pid})
			}
		}
		teams[i].Members = members
	}
}

// GetTeamInfo retrieves full team detail by ID, name, or ref_id.
// It calls /team/info for metadata, then /person/infos to resolve member names.
func (c *Client) GetTeamInfo(ctx context.Context, input *TeamGetInput) (*TeamItem, error) {
	infoBody := map[string]any{}
	if input.TeamID != 0 {
		infoBody["team_id"] = input.TeamID
	}
	if input.TeamName != "" {
		infoBody["team_name"] = input.TeamName
	}
	if input.RefID != "" {
		infoBody["ref_id"] = input.RefID
	}

	resp, err := c.makeRequest(ctx, "POST", "/team/info", infoBody)
	if err != nil {
		return nil, fmt.Errorf("unable to get team info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *TeamItem  `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, fmt.Errorf("team not found")
	}

	team := result.Data

	// Collect all person IDs that need enrichment: members + creator (if name missing).
	enrichIDs := make([]int64, 0, len(team.PersonIDs)+1)
	enrichIDs = append(enrichIDs, team.PersonIDs...)
	if team.CreatorID != 0 && team.CreatorName == "" {
		enrichIDs = append(enrichIDs, team.CreatorID)
	}

	if len(enrichIDs) > 0 {
		personMap, err := c.fetchPersonInfos(ctx, enrichIDs)
		if err != nil {
			c.logger.Warn("failed to enrich team members", "error", err)
		} else {
			members := make([]TeamMember, 0, len(team.PersonIDs))
			for _, pid := range team.PersonIDs {
				if p, ok := personMap[pid]; ok {
					members = append(members, TeamMember{
						PersonID:   p.PersonID,
						PersonName: p.PersonName,
						Email:      p.Email,
					})
				} else {
					members = append(members, TeamMember{PersonID: pid})
				}
			}
			team.Members = members

			if team.CreatorName == "" && team.CreatorID != 0 {
				if p, ok := personMap[team.CreatorID]; ok {
					team.CreatorName = p.PersonName
				}
			}
		}
	}

	return team, nil
}

// UpsertTeam creates or updates a team. TeamName is required by the API.
func (c *Client) UpsertTeam(ctx context.Context, input *TeamUpsertInput) (*TeamUpsertOutput, error) {
	if input.TeamName == "" {
		return nil, fmt.Errorf("team_name is required")
	}
	requestBody := map[string]any{
		"team_name": input.TeamName,
	}
	if input.TeamID != 0 {
		requestBody["team_id"] = input.TeamID
	}
	if input.Description != "" {
		requestBody["description"] = input.Description
	}
	if len(input.PersonIDs) > 0 {
		requestBody["person_ids"] = input.PersonIDs
	}
	if len(input.Emails) > 0 {
		requestBody["emails"] = input.Emails
	}
	if len(input.Phones) > 0 {
		requestBody["phones"] = input.Phones
	}
	if input.CountryCode != "" {
		requestBody["countryCode"] = input.CountryCode
	}
	if input.RefID != "" {
		requestBody["ref_id"] = input.RefID
	}
	if input.ResetIfNameExist {
		requestBody["reset_if_name_exist"] = true
	}

	resp, err := c.makeRequest(ctx, "POST", "/team/upsert", requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to upsert team: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError       `json:"error,omitempty"`
		Data  *TeamUpsertOutput `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, fmt.Errorf("unexpected empty response from team upsert")
	}

	return result.Data, nil
}

// DeleteTeam permanently deletes a team by ID, name, or ref_id.
func (c *Client) DeleteTeam(ctx context.Context, input *TeamDeleteInput) error {
	requestBody := map[string]any{}
	if input.TeamID != 0 {
		requestBody["team_id"] = input.TeamID
	}
	if input.TeamName != "" {
		requestBody["team_name"] = input.TeamName
	}
	if input.RefID != "" {
		requestBody["ref_id"] = input.RefID
	}

	resp, err := c.makeRequest(ctx, "POST", "/team/delete", requestBody)
	if err != nil {
		return fmt.Errorf("unable to delete team: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}
