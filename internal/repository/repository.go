package repository

import (
	"github.com/amirasaad/fintech/internal/account"
	"github.com/amirasaad/fintech/internal/model"
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
	var a model.Account
	result := r.db.First(&a, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return account.NewFromData(a.ID, a.Balance, a.Created, a.Updated), nil
}

func (r *accountRepository) Create(a *account.Account) error {
	result := r.db.Create(a)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Update(a *account.Account) error {
	dbModel := model.Account{
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
	result := r.db.Delete(&model.Account{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(transaction *account.Transaction) error {

	result := r.db.Create(transaction)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *transactionRepository) Get(id uuid.UUID) (*account.Transaction, error) {
	var t model.Transaction
	result := r.db.First(&t, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return account.NewTransactionFromData(t.ID, t.AccountID, t.Amount, t.Created), nil
}

func (r *transactionRepository) List(accountID uuid.UUID) ([]*account.Transaction, error) {
	var dbTransactions []*model.Transaction
	result := r.db.Where("account_id = ?", accountID).Find(&dbTransactions).Take(100)
	if result.Error != nil {
		return nil, result.Error
	}
	tx := make([]*account.Transaction, 0, len(dbTransactions))
	for _, t := range dbTransactions {
		tx = append(tx, account.NewTransactionFromData(t.ID, t.AccountID, t.Amount, t.Created))
	}
	return tx, nil
}
