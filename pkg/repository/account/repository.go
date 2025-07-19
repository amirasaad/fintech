package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/google/uuid"
)

// Repository defines the interface for account data access operations with support for CQRS (Command/Query Responsibility Segregation).
type Repository interface {
	// Create inserts a new account record from a DTO.
	Create(ctx context.Context, create dto.AccountCreate) error

	// Update updates an existing account by its ID using a DTO.
	Update(ctx context.Context, id uuid.UUID, update dto.AccountUpdate) error

	// Get retrieves an account by its ID as a read-optimized DTO.
	Get(ctx context.Context, id uuid.UUID) (*dto.AccountRead, error)

	// ListByUser lists all accounts for a given user as read-optimized DTOs.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*dto.AccountRead, error)
}
