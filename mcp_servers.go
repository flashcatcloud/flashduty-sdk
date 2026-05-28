package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

// CreateMCPServerInput is the payload for POST /safari/mcp/server/create.
// Transport must be one of "stdio", "sse", or "streamable-http". Fields are
// conditionally required by the backend depending on Transport: stdio uses
// Command/Args/Env; sse and streamable-http use URL/Headers. ConnectTimeout
// and CallTimeout are in seconds.
//
// TeamID is always serialized (no omitempty) because 0 is a meaningful
// sentinel — it explicitly requests account scope, distinct from "field
// omitted". Callers must set it deliberately: 0 = account scope; >0 = team scope.
type CreateMCPServerInput struct {
	ServerName     string            `json:"server_name"`
	Description    string            `json:"description"`
	Transport      string            `json:"transport"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	URL            string            `json:"url,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	ConnectTimeout int               `json:"connect_timeout,omitempty"`
	CallTimeout    int               `json:"call_timeout,omitempty"`
	Status         string            `json:"status,omitempty"`
	TeamID         int64             `json:"team_id"`
}

// CreateMCPServerOutput is the unwrapped data block returned by
// POST /safari/mcp/server/create.
type CreateMCPServerOutput struct {
	ServerID string `json:"server_id"`
	Status   string `json:"status"`
}

// CreateMCPServer registers a new MCP server with Flashduty.
func (c *Client) CreateMCPServer(ctx context.Context, input *CreateMCPServerInput) (*CreateMCPServerOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("create MCP server input is required")
	}

	resp, err := c.makeRequest(ctx, "POST", "/safari/mcp/server/create", input)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result struct {
		Error *DutyError             `json:"error,omitempty"`
		Data  *CreateMCPServerOutput `json:"data,omitempty"`
	}
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, fmt.Errorf("create MCP server returned empty data")
	}

	return result.Data, nil
}
