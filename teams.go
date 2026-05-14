package flashduty

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
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
					TeamID   int64  `json:"team_id"`
					TeamName string `json:"team_name"`
					Members  []struct {
						PersonID   int64  `json:"person_id"`
						PersonName string `json:"person_name"`
						Email      string `json:"email,omitempty"`
					} `json:"members,omitempty"`
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
				team := TeamInfo{
					TeamID:   t.TeamID,
					TeamName: t.TeamName,
				}
				if len(t.Members) > 0 {
					team.Members = make([]TeamMember, 0, len(t.Members))
					for _, m := range t.Members {
						team.Members = append(team.Members, TeamMember{
							PersonID:   m.PersonID,
							PersonName: m.PersonName,
							Email:      m.Email,
						})
					}
				}
				teams = append(teams, team)
			}
		}

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
				TeamID   int64  `json:"team_id"`
				TeamName string `json:"team_name"`
				Members  []struct {
					PersonID   int64  `json:"person_id"`
					PersonName string `json:"person_name"`
					Email      string `json:"email,omitempty"`
				} `json:"members,omitempty"`
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
			team := TeamInfo{
				TeamID:   t.TeamID,
				TeamName: t.TeamName,
			}
			if len(t.Members) > 0 {
				team.Members = make([]TeamMember, 0, len(t.Members))
				for _, m := range t.Members {
					team.Members = append(team.Members, TeamMember{
						PersonID:   m.PersonID,
						PersonName: m.PersonName,
						Email:      m.Email,
					})
				}
			}
			teams = append(teams, team)
		}
		total = result.Data.Total
	}

	return &ListTeamsOutput{
		Teams: teams,
		Total: total,
	}, nil
}

// GetTeamInfo retrieves full team detail by ID, name, or ref_id.
// It calls /team/info for full metadata and /team/infos for member names in parallel.
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

	var team *TeamItem
	var members []TeamMember

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		resp, err := c.makeRequest(gctx, "POST", "/team/info", infoBody)
		if err != nil {
			return fmt.Errorf("unable to get team info: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return handleAPIError(c.logger, resp)
		}

		var result struct {
			Error *DutyError `json:"error,omitempty"`
			Data  *TeamItem  `json:"data,omitempty"`
		}
		if err := parseResponse(c.logger, resp, &result); err != nil {
			return err
		}
		if result.Error != nil {
			return result.Error
		}
		if result.Data == nil {
			return fmt.Errorf("team not found")
		}
		team = result.Data
		return nil
	})

	// When looking up by ID, we can fire /team/infos in parallel for member names.
	// For name/ref_id lookups we don't know the ID upfront, so we enrich after.
	if input.TeamID != 0 {
		g.Go(func() error {
			resolved, err := c.fetchTeamMembers(gctx, input.TeamID)
			if err != nil {
				c.logger.Warn("failed to enrich team members", "error", err)
				return nil
			}
			members = resolved
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// For name/ref_id lookups, enrich members sequentially using the team_id we got back.
	if input.TeamID == 0 && team != nil && len(team.PersonIDs) > 0 {
		resolved, err := c.fetchTeamMembers(ctx, team.TeamID)
		if err != nil {
			c.logger.Warn("failed to enrich team members", "error", err)
		} else {
			members = resolved
		}
	}

	if len(members) > 0 {
		team.Members = members
	}

	return team, nil
}

// fetchTeamMembers retrieves member details for a team via /team/infos.
func (c *Client) fetchTeamMembers(ctx context.Context, teamID int64) ([]TeamMember, error) {
	infosBody := map[string]any{"team_ids": []int64{teamID}}
	resp, err := c.makeRequest(ctx, "POST", "/team/infos", infosBody)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch team members: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError `json:"error,omitempty"`
		Data  *struct {
			Items []struct {
				Members []struct {
					PersonID   int64  `json:"person_id"`
					PersonName string `json:"person_name"`
					Email      string `json:"email,omitempty"`
				} `json:"members,omitempty"`
			} `json:"items"`
		} `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	var members []TeamMember
	if result.Data != nil && len(result.Data.Items) > 0 {
		for _, m := range result.Data.Items[0].Members {
			members = append(members, TeamMember{
				PersonID:   m.PersonID,
				PersonName: m.PersonName,
				Email:      m.Email,
			})
		}
	}
	return members, nil
}

// UpsertTeam creates or updates a team.
func (c *Client) UpsertTeam(ctx context.Context, input *TeamUpsertInput) (*TeamUpsertOutput, error) {
	requestBody := map[string]any{}
	if input.TeamName != "" {
		requestBody["team_name"] = input.TeamName
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
