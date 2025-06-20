package repository

import (
	"github.com/amirasaad/fintech/internal/account"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountRepository interface {
	Get(id uuid.UUID) (*account.Account, error)
	Create(account *account.Account) error
	Update(account *account.Account) error
	Delete(id uuid.UUID) error
}

type TransactionRepository interface {
	Create(transaction *account.Transaction) error
	Get(id uuid.UUID) (*account.Transaction, error)
	List(accountID uuid.UUID) ([]*account.Transaction, error)
}

type accountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Get(id uuid.UUID) (*account.Account, error) {
	var account account.Account
	result := r.db.First(&account, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &account, nil
}

func (r *accountRepository) Create(account *account.Account) error {
	result := r.db.Create(account)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Update(account *account.Account) error {
	result := r.db.Save(account)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&account.Account{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
