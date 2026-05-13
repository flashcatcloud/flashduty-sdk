package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

// AccountInfo contains account details returned by the account info API.
type AccountInfo struct {
	AccountID   uint64 `json:"account_id"`
	AccountName string `json:"account_name"`
	Domain      string `json:"domain"`
	Email       string `json:"email"`
	Phone       string `json:"phone,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	Locale      string `json:"locale,omitempty"`
	TimeZone    string `json:"time_zone,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

type accountInfoResponse struct {
	Error *DutyError   `json:"error,omitempty"`
	Data  *AccountInfo `json:"data,omitempty"`
}

// GetAccountInfo retrieves the account information for the authenticated app key.
func (c *Client) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	resp, err := c.makeRequest(ctx, "POST", "/account/info", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("unable to get account info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result accountInfoResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, fmt.Errorf("empty account info in response")
	}

	return result.Data, nil
}
