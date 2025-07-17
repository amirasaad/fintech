package account

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	repo "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

// Create implements account.Repository.
func (r *repository) Create(ctx context.Context, create dto.AccountCreate) error {
	panic("unimplemented")
}

// Get implements account.Repository.
func (r *repository) Get(ctx context.Context, id uuid.UUID) (*dto.AccountRead, error) {
	panic("unimplemented")
}

// ListByUser implements account.Repository.
func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*dto.AccountRead, error) {
	panic("unimplemented")
}

// Update implements account.Repository.
func (r *repository) Update(ctx context.Context, id uuid.UUID, update dto.AccountUpdate) error {
	panic("unimplemented")
}

// NewRepository creates a new CQRS-style account repository using the provided *gorm.DB.
func NewRepository(db *gorm.DB) repo.Repository {
	return &repository{db: db}
}
