package infrarepo // import alias for infra/repository/transaction

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	repo "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new CQRS-style transaction repository using the provided *gorm.DB.
func New(db *gorm.DB) repo.Repository {
	return &repository{db: db}
}

// Create implements transaction.Repository.
func (r *repository) Create(ctx context.Context, create dto.TransactionCreate) error {
	panic("unimplemented")
}

// Get implements transaction.Repository.
func (r *repository) Get(ctx context.Context, id uuid.UUID) (*dto.TransactionRead, error) {
	panic("unimplemented")
}

// GetByPaymentID implements transaction.Repository.
func (r *repository) GetByPaymentID(ctx context.Context, paymentID string) (*dto.TransactionRead, error) {
	panic("unimplemented")
}

// ListByAccount implements transaction.Repository.
func (r *repository) ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*dto.TransactionRead, error) {
	panic("unimplemented")
}

// ListByUser implements transaction.Repository.
func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*dto.TransactionRead, error) {
	panic("unimplemented")
}

// PartialUpdate implements transaction.Repository.
func (r *repository) PartialUpdate(ctx context.Context, id uuid.UUID, update dto.TransactionUpdate) error {
	panic("unimplemented")
}

// Update implements transaction.Repository.
func (r *repository) Update(ctx context.Context, id uuid.UUID, update dto.TransactionUpdate) error {
	panic("unimplemented")
}

// UpsertByPaymentID implements transaction.Repository.
func (r *repository) UpsertByPaymentID(ctx context.Context, paymentID string, create dto.TransactionCreate) error {
	panic("unimplemented")
}
