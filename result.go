package verify

import "time"

// Result holds the outcome of a single email verification.
type Result struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Valid      bool      `json:"valid"`
	MXFound    bool      `json:"mx_found"`
	Disposable bool      `json:"disposable"`
	RoleBased  bool      `json:"role_based"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// HistoryPage is returned by ListHistory.
type HistoryPage struct {
	Verifications []*Result `json:"verifications"`
	Limit         int       `json:"limit"`
	Offset        int       `json:"offset"`
}
