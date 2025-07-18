package account

import (
	"context"

	model "github.com/amirasaad/fintech/infra/repository/model" // domain model import
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	repo "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new CQRS-style account repository using the provided *gorm.DB.
func New(db *gorm.DB) repo.Repository {
	return &repository{db: db}
}

// Create implements account.Repository.
func (r *repository) Create(ctx context.Context, create dto.AccountCreate) error {
	acct := mapCreateDTOToModel(create)
	return r.db.WithContext(ctx).Create(&acct).Error
}

// Update implements account.Repository.
func (r *repository) Update(ctx context.Context, id uuid.UUID, update dto.AccountUpdate) error {
	updates := mapUpdateDTOToModel(update)
	return r.db.WithContext(ctx).Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

// Get implements account.Repository.
func (r *repository) Get(ctx context.Context, id uuid.UUID) (*dto.AccountRead, error) {
	var acct model.Account
	if err := r.db.WithContext(ctx).First(&acct, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return mapModelToDTO(&acct), nil
}

// ListByUser implements account.Repository.
func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*dto.AccountRead, error) {
	var accts []model.Account
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&accts).Error; err != nil {
		return nil, err
	}
	result := make([]*dto.AccountRead, 0, len(accts))
	for i := range accts {
		result = append(result, mapModelToDTO(&accts[i]))
	}
	return result, nil
}

// mapCreateDTOToModel maps AccountCreate DTO to GORM model.
func mapCreateDTOToModel(create dto.AccountCreate) model.Account {
	return model.Account{
		ID:       create.ID,
		UserID:   create.UserID,
		Balance:  0,
		Currency: create.Currency,
		// Add more fields as needed
	}
}

// mapUpdateDTOToModel maps AccountUpdate DTO to a map for GORM Updates.
func mapUpdateDTOToModel(update dto.AccountUpdate) map[string]any {
	updates := make(map[string]any)
	if update.Balance != nil {
		updates["balance"] = *update.Balance
	}
	// if update.Status != nil {
	// 	updates["status"] = *update.Status
	// }
	// Add more fields as needed
	return updates
}

// mapModelToDTO maps a GORM model to a read-optimized DTO.
func mapModelToDTO(acct *model.Account) *dto.AccountRead {
	bal := money.NewFromData(acct.Balance, acct.Currency)
	return &dto.AccountRead{
		ID:        acct.ID,
		UserID:    acct.UserID,
		Balance:   bal.AmountFloat(),
		Currency:  bal.Currency().String(),
		CreatedAt: acct.CreatedAt,

		// Status: acct.Status, // Uncomment if Status exists in model.Account
		// Add more fields as needed
	}
}
