package verify

import (
	"encoding/json"
	"fmt"
)

// APIError is returned when the Tickstem API responds with a 4xx or 5xx status.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("tickstem/verify: API error %d: %s", e.StatusCode, e.Message)
}

func IsUnauthorized(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.StatusCode == 401
}

// IsQuotaExceeded reports whether the monthly verification quota has been reached.
func IsQuotaExceeded(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.StatusCode == 402
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

func parseAPIError(statusCode int, body []byte) *APIError {
	var errResp apiErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return &APIError{StatusCode: statusCode, Message: errResp.Error}
	}
	return &APIError{StatusCode: statusCode, Message: string(body)}
}
