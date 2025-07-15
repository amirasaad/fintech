package account

import "github.com/google/uuid"

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
	EventID       uuid.UUID         // Unique event ID (UUID)
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

// EventType returns the type of the PaymentEvent.
func (e PaymentEvent) EventType() string { return "PaymentEvent" }

// DepositRequestedEvent is emitted when a deposit is requested (pure event-driven domain).
type DepositRequestedEvent struct {
	EventID   uuid.UUID
	AccountID string
	UserID    string
	Amount    float64
	Currency  string
	Source    MoneySource
	Timestamp int64
}

// EventType returns the type of the DepositRequestedEvent.
func (e DepositRequestedEvent) EventType() string { return "DepositRequestedEvent" }

// WithdrawRequestedEvent is emitted when a withdrawal is requested (pure event-driven domain).
type WithdrawRequestedEvent struct {
	EventID   uuid.UUID
	AccountID string
	UserID    string
	Amount    float64
	Currency  string
	Target    ExternalTarget
	Timestamp int64
}

// EventType returns the type of the WithdrawRequestedEvent.
func (e WithdrawRequestedEvent) EventType() string { return "WithdrawRequestedEvent" }

// TransferRequestedEvent is emitted when a transfer is requested (pure event-driven domain).
type TransferRequestedEvent struct {
	EventID         uuid.UUID
	SourceAccountID uuid.UUID
	DestAccountID   uuid.UUID
	SenderUserID    uuid.UUID
	ReceiverUserID  uuid.UUID
	Amount          float64
	Currency        string
	Source          MoneySource
	Timestamp       int64
}

// EventType returns the type of the TransferRequestedEvent.
func (e TransferRequestedEvent) EventType() string { return "TransferRequestedEvent" }
