package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

const defaultTeamsQueryLimit = 20

// ListTeamsInput contains parameters for listing teams
type ListTeamsInput struct {
	TeamIDs []int64 // Direct lookup by team IDs
	Name    string  // Search by team name
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
	requestBody := map[string]any{
		"p":     1,
		"limit": defaultTeamsQueryLimit,
	}
	if input.Name != "" {
		requestBody["team_name"] = input.Name
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
