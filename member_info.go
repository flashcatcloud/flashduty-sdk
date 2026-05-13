package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

type MemberInfo struct {
	AccountID   uint64 `json:"account_id"`
	AccountName string `json:"account_name"`
	MemberID    uint64 `json:"member_id"`
	MemberName  string `json:"member_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	Locale      string `json:"locale,omitempty"`
	TimeZone    string `json:"time_zone,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

type memberInfoResponse struct {
	Error *DutyError  `json:"error,omitempty"`
	Data  *MemberInfo `json:"data,omitempty"`
}

func (c *Client) GetMemberInfo(ctx context.Context) (*MemberInfo, error) {
	resp, err := c.makeRequest(ctx, "POST", "/member/info", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("unable to get member info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result memberInfoResponse
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if result.Data == nil {
		return nil, fmt.Errorf("empty member info in response")
	}

	return result.Data, nil
}
