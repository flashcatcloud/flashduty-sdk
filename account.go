package flashduty

import (
	"context"
	"fmt"
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

// GetAccountInfo retrieves the account information for the authenticated app key.
func (c *Client) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	data, err := postOptionalData[AccountInfo](c, ctx, "/account/info", map[string]any{}, "unable to get account info")
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("empty account info in response")
	}

	return data, nil
}
