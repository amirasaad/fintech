package dto

import (
	"time"

	"github.com/google/uuid"
)

// TransactionRead is a read-optimized DTO for transaction queries, API responses, and reporting.
type TransactionRead struct {
	ID        uuid.UUID // Unique transaction identifier
	UserID    uuid.UUID // User who owns the transaction
	AccountID uuid.UUID // Account associated with the transaction
	Amount    float64   // Transaction amount (use string for high precision if needed)
	Status    string    // Transaction status (e.g., completed, pending)
	PaymentID string    // External payment provider ID
	CreatedAt time.Time // Timestamp of transaction creation
	// Add audit, denormalized, or computed fields as needed
}

// TransactionCreate is a DTO for creating a new transaction.
type TransactionCreate struct {
	ID          uuid.UUID
	UserID      uuid.UUID // User who owns the transaction
	AccountID   uuid.UUID // Account associated with the transaction
	Amount      int64     // Transaction amount
	Status      string    // Initial status
	Currency    string
	MoneySource string
	// Add more fields as needed for creation
}

// TransactionUpdate is a DTO for updating one or more fields of a transaction.
type TransactionUpdate struct {
	Status    *string // Optional status update
	PaymentID *string // Optional payment provider ID update
	// Add more fields as needed for partial updates
}

// TransactionCommand is a DTO for user/service input (main unit, float64).
type TransactionCommand struct {
	UserID      uuid.UUID // User who owns the transaction
	AccountID   uuid.UUID // Account associated with the transaction
	Amount      float64   // Main unit (e.g., dollars)
	Currency    string
	MoneySource string
	// Add more fields as needed for commands
}
