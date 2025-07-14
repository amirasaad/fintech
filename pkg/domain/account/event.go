package account

// PaymentStatus represents the status of a payment transaction event.
type PaymentStatus string

const (
	// PaymentStatusInitiated indicates the user requested a payment (not yet sent to provider).
	PaymentStatusInitiated PaymentStatus = "initiated"
	// PaymentStatusPending indicates the payment is in progress (sent to provider, awaiting confirmation).
	PaymentStatusPending PaymentStatus = "pending"
	// PaymentStatusCompleted indicates the payment has been confirmed and completed.
	PaymentStatusCompleted PaymentStatus = "completed"
	// PaymentStatusFailed indicates the payment has failed or was rejected.
	PaymentStatusFailed PaymentStatus = "failed"
)

// PaymentEvent represents an event in the payment lifecycle (initiated, pending, completed, failed).
type PaymentEvent struct {
	EventID       string            // Unique event ID (UUID)
	TransactionID string            // Associated transaction
	AccountID     string            // Account involved
	UserID        string            // User who initiated
	Amount        int64             // Amount in minor units
	Currency      string            // Currency code (ISO 4217)
	Status        PaymentStatus     // initiated, pending, completed, failed
	Provider      string            // Payment provider name
	Timestamp     int64             // Unix timestamp (UTC)
	Metadata      map[string]string // Optional extra info
}
