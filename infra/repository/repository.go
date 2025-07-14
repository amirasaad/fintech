package repository

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository creates a new account repository instance.
func NewAccountRepository(db *gorm.DB) repository.AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Get(id uuid.UUID) (*account.Account, error) {
	var a Account
	result := r.db.First(&a, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, account.ErrAccountNotFound
		}
		return nil, result.Error
	}
	accBalance := money.NewFromData(a.Balance, a.Currency)
	return account.NewAccountFromData(a.ID, a.UserID, accBalance, a.CreatedAt, a.UpdatedAt), nil
}

func (r *accountRepository) Create(acc *account.Account) error {
	a := Account{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		ID:       acc.ID,
		UserID:   acc.UserID,
		Balance:  acc.Balance.Amount(),
		Currency: string(acc.Balance.Currency()),
	}
	result := r.db.Create(&a)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Update(a *account.Account) error {
	// Use infra.Account for DB operations
	dbModel := Account{
		Model: gorm.Model{
			CreatedAt: a.CreatedAt,
			DeletedAt: gorm.DeletedAt{},
			UpdatedAt: time.Now().UTC(),
		},
		ID:       a.ID,
		UserID:   a.UserID,
		Balance:  a.Balance.Amount(),
		Currency: string(a.Balance.Currency()),
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

// NewTransactionRepository creates a new transaction repository instance.
func NewTransactionRepository(db *gorm.DB) repository.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(transaction *account.Transaction, convInfo *common.ConversionInfo, externalTargetMasked string) error {
	// Convert domain transaction to GORM model
	dbTransaction := Transaction{
		Model: gorm.Model{
			CreatedAt: transaction.CreatedAt,
			UpdatedAt: transaction.CreatedAt,
		},
		ID:                   transaction.ID,
		AccountID:            transaction.AccountID,
		UserID:               transaction.UserID,
		Amount:               transaction.Amount.Amount(),
		Currency:             string(transaction.Amount.Currency()),
		Balance:              transaction.Balance.Amount(),
		MoneySource:          string(transaction.MoneySource),
		ExternalTargetMasked: externalTargetMasked,
	}

	// Only set conversion fields if conversion info is provided
	if convInfo != nil {
		dbTransaction.OriginalAmount = &convInfo.OriginalAmount
		dbTransaction.OriginalCurrency = &convInfo.OriginalCurrency
		dbTransaction.ConversionRate = &convInfo.ConversionRate
	}

	result := r.db.Create(&dbTransaction)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *transactionRepository) Get(
	id uuid.UUID,
) (
	*account.Transaction,
	error,
) {
	var t Transaction
	result := r.db.First(&t, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, account.ErrAccountNotFound
		}
		return nil, result.Error
	}
	amount := money.NewFromData(t.Balance, t.Currency)
	balance := money.NewFromData(t.Amount, t.Currency)
	return account.NewTransactionFromData(
		t.ID,
		t.UserID, t.AccountID, amount, balance, account.MoneySource(t.MoneySource), t.CreatedAt), nil
}

func (r *transactionRepository) List(
	userID, accountID uuid.UUID,
) ([]*account.Transaction, error) {
	var dbTransactions []*Transaction
	result := r.db.Where("account_id = ? and user_id = ?", accountID, userID).Order("created_at desc").Limit(100).Find(&dbTransactions)
	if result.Error != nil {
		return nil, result.Error
	}
	tx := make([]*account.Transaction, 0, len(dbTransactions))
	for _, t := range dbTransactions {
		amount := money.NewFromData(t.Balance, t.Currency)
		balance := money.NewFromData(t.Amount, t.Currency)
		tx = append(tx, account.NewTransactionFromData(t.ID, t.UserID, t.AccountID, amount, balance, account.MoneySource(t.MoneySource), t.CreatedAt))
	}
	return tx, nil
}

type userRepository struct {
	db *gorm.DB
}

// Valid implements repository.UserRepository.
func (u *userRepository) Valid(id uuid.UUID, password string) bool {
	var usr user.User
	result := u.db.Where("id = ?", id).First(&usr)
	if result.Error != nil {
		return false
	}
	// Compare the provided password with the stored hash
	return utils.CheckPasswordHash(password, usr.Password)
}

// Create implements repository.UserRepository.
func (u *userRepository) Create(user *user.User) error {
	result := u.db.Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Delete implements repository.UserRepository.
func (u *userRepository) Delete(id uuid.UUID) error {
	result := u.db.Delete(&user.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Get implements repository.UserRepository.
func (u *userRepository) Get(id uuid.UUID) (*user.User, error) {
	var usr user.User
	result := u.db.First(&usr, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, result.Error
	}
	return &usr, nil
}

// GetByEmail implements repository.UserRepository.
func (u *userRepository) GetByEmail(email string) (*user.User, error) {
	var usr user.User
	result := u.db.Where("email = ?", email).First(&usr)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, result.Error
	}
	return &usr, nil
}

// GetByUsername implements repository.UserRepository.
func (u *userRepository) GetByUsername(username string) (*user.User, error) {
	var usr user.User
	result := u.db.Where("username = ?", username).First(&usr)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, result.Error
	}
	return &usr, nil
}

// Update implements repository.UserRepository.
func (u *userRepository) Update(user *user.User) error {
	result := u.db.Save(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// NewUserRepository creates a new user repository instance.
func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{db: db}
}
