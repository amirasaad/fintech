package dto

import (
	"time"

	"github.com/google/uuid"
)

// TransactionRead is a read-optimized DTO for transaction queries, API responses, and reporting.
type TransactionRead struct {
	ID              uuid.UUID // Unique transaction identifier
	UserID          uuid.UUID // User who owns the transaction
	AccountID       uuid.UUID // Account associated with the transaction
	Amount          float64   // Transaction amount (use string for high precision if needed)
	Currency        string    // Transaction currency
	Balance         float64   // Account balance after transaction
	Status          string    // Transaction status (e.g., completed, pending)
	PaymentID       *string   // External payment provider ID
	CreatedAt       time.Time // Timestamp of transaction creation
	Fee             float64   // Total transaction fee
	ConvertedAmount float64   // Converted amount after conversion
	TargetCurrency  string    // Target currency after conversion
	// Add audit, denormalized, or computed fields as needed
}

// TransactionCreate is a DTO for creating a new transaction.
type TransactionCreate struct {
	ID        uuid.UUID
	UserID    uuid.UUID // User who owns the transaction
	AccountID uuid.UUID // Account associated with the transaction
	// External payment provider ID (pointer to allow NULL in database)
	PaymentID            *string
	Amount               int64  // Transaction amount
	Status               string // Initial status
	Currency             string
	MoneySource          string
	ExternalTargetMasked string
	TargetCurrency       string
	Fee                  int64 // Total transaction fee
	// Add more fields as needed for creation
}

// TransactionUpdate is a DTO for updating one or more fields of a transaction.
type TransactionUpdate struct {
	Status    *string // Optional status update
	PaymentID *string // Optional payment provider ID update
	// Conversion fields (nullable when no conversion occurs)
	Balance          *int64
	Amount           *int64
	Currency         *string
	OriginalAmount   *float64
	OriginalCurrency *string
	ConvertedAmount  *float64
	ConversionRate   *float64
	TargetCurrency   *string
	// Add more fields as needed for partial updates
	Fee *int64
}
