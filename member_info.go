package flashduty

import (
	"context"
	"fmt"
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

func (c *Client) GetMemberInfo(ctx context.Context) (*MemberInfo, error) {
	data, err := postOptionalData[MemberInfo](c, ctx, "/member/info", map[string]any{}, "unable to get member info")
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("empty member info in response")
	}

	return data, nil
}
