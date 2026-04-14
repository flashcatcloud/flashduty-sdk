package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

const defaultMembersQueryLimit = 20

// ListMembersInput contains parameters for listing members
type ListMembersInput struct {
	PersonIDs []int64 // Direct lookup by person IDs
	Name      string  // Search by member name (fuzzy match)
	Email     string  // Search by email address
	Page      int     // Page number (default 1)
}

// ListMembersOutput contains the result of listing members
type ListMembersOutput struct {
	// PersonInfos is populated when querying by PersonIDs
	PersonInfos []PersonInfo `json:"person_infos,omitempty"`
	// Members is populated when listing/searching members
	Members []MemberItem `json:"members,omitempty"`
	Total   int          `json:"total"`
}

// ListMembers queries members by IDs, name, or email
func (c *Client) ListMembers(ctx context.Context, input *ListMembersInput) (*ListMembersOutput, error) {
	// Query by person IDs
	if len(input.PersonIDs) > 0 {
		personMap, err := c.fetchPersonInfos(ctx, input.PersonIDs)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve members: %w", err)
		}

		members := make([]PersonInfo, 0, len(personMap))
		for _, p := range personMap {
			members = append(members, p)
		}

		return &ListMembersOutput{
			PersonInfos: members,
			Total:       len(members),
		}, nil
	}

	// List all members with optional filters
	page := input.Page
	if page <= 0 {
		page = 1
	}
	requestBody := map[string]any{
		"p":     page,
		"limit": defaultMembersQueryLimit,
	}
	// The API expects a single "query" field for fuzzy search.
	// Name takes priority over Email when both are provided.
	if input.Name != "" {
		requestBody["query"] = input.Name
	} else if input.Email != "" {
		requestBody["query"] = input.Email
	}

	resp, err := c.makeRequest(ctx, "POST", "/member/list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to list members: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result MemberListResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}

	members := []MemberItem{}
	total := 0
	if result.Data != nil {
		members = result.Data.Items
		total = result.Data.Total
	}

	return &ListMembersOutput{
		Members: members,
		Total:   total,
	}, nil
}
