package user

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/google/uuid"
)

// Repository defines the interface for user data access operations with
// support for CQRS (Command/Query Responsibility Segregation).
type Repository interface {
	// Create inserts a new user record from a DTO.
	Create(ctx context.Context, create *dto.UserCreate) error

	// Update updates an existing user by its ID using a DTO.
	Update(ctx context.Context, id uuid.UUID, update *dto.UserUpdate) error

	// Get retrieves a user by its ID as a read-optimized DTO.
	Get(ctx context.Context, id uuid.UUID) (*dto.UserRead, error)

	// GetByEmail retrieves a user by email as a read-optimized DTO.
	GetByEmail(ctx context.Context, email string) (*dto.UserRead, error)

	// GetByUsername retrieves a user by username as a read-optimized DTO.
	GetByUsername(ctx context.Context, username string) (*dto.UserRead, error)

	// Delete deletes a user by its ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all users as read-optimized DTOs with pagination support.
	List(ctx context.Context, page, pageSize int) ([]*dto.UserRead, error)

	// Exists checks if a user with the given ID exists.
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// ExistsByEmail checks if a user with the given email exists.
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByUsername checks if a user with the given username exists.
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}
