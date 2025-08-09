package transaction

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/google/uuid"
)

// Repository defines the interface for transaction data
// access operations with support for CQRS (Command/Query Responsibility Segregation).
type Repository interface {
	// Create inserts a new transaction record from a DTO.
	Create(ctx context.Context, create dto.TransactionCreate) error

	// Update updates an existing transaction by its ID using a DTO.
	Update(ctx context.Context, id uuid.UUID, update dto.TransactionUpdate) error

	// PartialUpdate updates specified fields of a transaction by its ID using a DTO.
	PartialUpdate(ctx context.Context, id uuid.UUID, update dto.TransactionUpdate) error

	// Upsert inserts or updates a transaction by a business key (e.g., event ID, payment ID).
	UpsertByPaymentID(ctx context.Context, paymentID string, create dto.TransactionCreate) error

	// Get retrieves a transaction by its ID as a read-optimized DTO.
	Get(ctx context.Context, id uuid.UUID) (*dto.TransactionRead, error)

	// GetByPaymentID retrieves a transaction by its payment provider ID as a read-optimized DTO.
	GetByPaymentID(ctx context.Context, paymentID string) (*dto.TransactionRead, error)

	// ListByUser lists all transactions for a given user as read-optimized DTOs.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*dto.TransactionRead, error)

	// ListByAccount lists all transactions for a given account as read-optimized DTOs.
	ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*dto.TransactionRead, error)
}
