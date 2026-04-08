package flashduty

import "fmt"

// FlashdutyResponse represents the standard Flashduty API response structure
type FlashdutyResponse struct {
	Error *DutyError  `json:"error,omitempty"`
	Data  any `json:"data,omitempty"`
}

// DutyError represents Flashduty API error
type DutyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface for DutyError
func (e *DutyError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
