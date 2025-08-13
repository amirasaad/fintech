package checkout

import "time"

// SessionDTO represents a checkout session for API responses.
type SessionDTO struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transaction_id"`
	UserID        string    `json:"user_id"`
	AccountID     string    `json:"account_id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CheckoutURL   string    `json:"checkout_url"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}
