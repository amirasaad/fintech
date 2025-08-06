package dto

import (
	"time"

	"github.com/google/uuid"
)

// AccountRead is a read-optimized DTO for account queries, API responses, and reporting.
type AccountRead struct {
	ID        uuid.UUID // Unique account identifier
	UserID    uuid.UUID // User who owns the account
	Balance   float64   // Account balance
	Currency  string
	Status    string    // Account status (e.g., active, closed)
	CreatedAt time.Time // Timestamp of account creation
	// Add more fields as needed for queries
}

// AccountCreate is a DTO for creating a new account.
type AccountCreate struct {
	ID       uuid.UUID
	UserID   uuid.UUID // User who owns the account
	Balance  int64     // Initial balance
	Status   string    // Initial status
	Currency string
	// Add more fields as needed for creation
}

// AccountUpdate is a DTO for updating one or more fields of an account.
type AccountUpdate struct {
	Balance *int64  // Optional balance update
	Status  *string // Optional status update
	// Add more fields as needed for partial updates
}
