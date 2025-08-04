package infrarepo // import alias for infra/repository/transaction

import (
	"context"

	"github.com/amirasaad/fintech/pkg/dto"
	repo "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/amirasaad/fintech/infra/repository/model"
)

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new CQRS-style transaction repository using the provided *gorm.DB.
func New(db *gorm.DB) repo.Repository {
	return &repository{db: db}
}

// Create implements transaction.Repository.
func (r *repository) Create(
	ctx context.Context,
	create dto.TransactionCreate,
) error {
	tx := mapCreateDTOToModel(create)
	return r.db.WithContext(ctx).Create(&tx).Error
}

// Update implements transaction.Repository.
func (r *repository) Update(
	ctx context.Context,
	id uuid.UUID,
	update dto.TransactionUpdate,
) error {
	updates := mapUpdateDTOToModel(update)
	return r.db.WithContext(
		ctx,
	).Model(
		&model.Transaction{},
	).Where(
		"id = ?",
		id,
	).Updates(
		updates,
	).Error
}

// PartialUpdate implements transaction.Repository.
func (r *repository) PartialUpdate(
	ctx context.Context,
	id uuid.UUID,
	update dto.TransactionUpdate,
) error {
	updates := mapUpdateDTOToModel(update)
	return r.db.WithContext(
		ctx,
	).Model(
		&model.Transaction{},
	).Where(
		"id = ?",
		id,
	).Updates(
		updates,
	).Error
}

// UpsertByPaymentID implements transaction.Repository.
func (r *repository) UpsertByPaymentID(
	ctx context.Context,
	paymentID string,
	create dto.TransactionCreate,
) error {
	tx := mapCreateDTOToModel(create)
	tx.PaymentID = paymentID
	return r.db.WithContext(
		ctx,
	).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "payment_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "amount"}),
		},
	).Create(&tx).Error
}

// Get implements transaction.Repository.
func (r *repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (*dto.TransactionRead, error) {
	var tx model.Transaction
	if err := r.db.WithContext(
		ctx,
	).First(
		&tx,
		"id = ?",
		id,
	).Error; err != nil {
		return nil, err
	}
	return mapModelToReadDTO(&tx), nil
}

// GetByPaymentID implements transaction.Repository.
func (r *repository) GetByPaymentID(
	ctx context.Context,
	paymentID string,
) (*dto.TransactionRead, error) {
	var tx model.Transaction
	if err := r.db.WithContext(
		ctx,
	).Where(
		"payment_id = ?",
		paymentID,
	).First(
		&tx,
	).Error; err != nil {
		return nil, err
	}
	return mapModelToReadDTO(&tx), nil
}

// ListByUser implements transaction.Repository.
func (r *repository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]*dto.TransactionRead, error) {
	var txs []model.Transaction
	if err := r.db.WithContext(
		ctx,
	).Where(
		"user_id = ?",
		userID,
	).Find(
		&txs,
	).Error; err != nil {
		return nil, err
	}
	result := make([]*dto.TransactionRead, 0, len(txs))
	for i := range txs {
		result = append(result, mapModelToReadDTO(&txs[i]))
	}
	return result, nil
}

// ListByAccount implements transaction.Repository.
func (r *repository) ListByAccount(
	ctx context.Context,
	accountID uuid.UUID,
) ([]*dto.TransactionRead, error) {
	var txs []model.Transaction
	if err := r.db.WithContext(
		ctx,
	).Where(
		"account_id = ?",
		accountID,
	).Find(
		&txs,
	).Error; err != nil {
		return nil, err
	}
	result := make([]*dto.TransactionRead, 0, len(txs))
	for i := range txs {
		result = append(result, mapModelToReadDTO(&txs[i]))
	}
	return result, nil
}

// --- Mappers ---

func mapCreateDTOToModel(create dto.TransactionCreate) model.Transaction {
	return model.Transaction{
		ID:          create.ID,
		UserID:      create.UserID,
		AccountID:   create.AccountID,
		Amount:      create.Amount,
		Status:      create.Status,
		MoneySource: create.MoneySource,
		// Add more fields as needed
	}
}

func mapUpdateDTOToModel(update dto.TransactionUpdate) map[string]any {
	updates := make(map[string]any)
	if update.Status != nil {
		updates["status"] = *update.Status
	}
	if update.PaymentID != nil {
		updates["payment_id"] = *update.PaymentID
	}
	if update.ConversionRate != nil {
		updates["conversion_rate"] = update.ConversionRate
	}
	if update.OriginalAmount != nil {
		updates["original_amount"] = update.OriginalAmount
	}
	if update.OriginalCurrency != nil {
		updates["original_currency"] = *update.OriginalCurrency
	}

	// Add more fields as needed
	return updates
}

func mapModelToReadDTO(tx *model.Transaction) *dto.TransactionRead {
	return &dto.TransactionRead{
		ID:        tx.ID,
		UserID:    tx.UserID,
		AccountID: tx.AccountID,
		Amount:    float64(tx.Amount),
		Status:    tx.Status,
		PaymentID: tx.PaymentID,
		CreatedAt: tx.CreatedAt,
		// Add more fields as needed
	}
}
