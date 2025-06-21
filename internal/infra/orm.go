package infra

import (
	"github.com/amirasaad/fintech/internal/domain"
	"github.com/amirasaad/fintech/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type accountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) repository.AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Get(id uuid.UUID) (*domain.Account, error) {
	var a Account
	result := r.db.First(&a, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return domain.NewAccountFromData(a.ID, a.Balance, a.Created, a.Updated), nil
}

func (r *accountRepository) Create(a *domain.Account) error {
	result := r.db.Create(a)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Update(a *domain.Account) error {
	dbModel := Account{
		ID:      a.ID,
		Balance: a.Balance,
	}
	result := r.db.Save(&dbModel)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&Account{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) repository.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(transaction *domain.Transaction) error {

	result := r.db.Create(transaction)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *transactionRepository) Get(id uuid.UUID) (*domain.Transaction, error) {
	var t Transaction
	result := r.db.First(&t, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return domain.NewTransactionFromData(t.ID, t.AccountID, t.Amount, t.Balance, t.Created), nil
}

func (r *transactionRepository) List(accountID uuid.UUID) ([]*domain.Transaction, error) {
	var dbTransactions []*Transaction
	result := r.db.Where("account_id = ?", accountID).Order("created desc").Limit(100).Find(&dbTransactions)
	if result.Error != nil {
		return nil, result.Error
	}
	tx := make([]*domain.Transaction, 0, len(dbTransactions))
	for _, t := range dbTransactions {
		tx = append(tx, domain.NewTransactionFromData(t.ID, t.AccountID, t.Amount, t.Balance, t.Created))
	}
	return tx, nil
}
