package repository

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type accountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) repository.AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Get(id uuid.UUID) (*account.Account, error) {
	var a account.Account
	result := r.db.First(&a, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, account.ErrAccountNotFound
		}
		return nil, result.Error
	}
	return account.NewAccountFromData(a.ID, a.UserID, a.Balance, a.Currency, a.CreatedAt, a.UpdatedAt), nil
}

func (r *accountRepository) Create(a *account.Account) error {
	result := r.db.Create(a)
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
		Balance:  a.Balance,
		Currency: string(a.Currency),
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

func (r *transactionRepository) Create(transaction *account.Transaction) error {
	// Convert domain transaction to GORM model
	dbTransaction := Transaction{
		Model: gorm.Model{
			CreatedAt: transaction.CreatedAt,
			UpdatedAt: transaction.CreatedAt,
		},
		ID:               transaction.ID,
		AccountID:        transaction.AccountID,
		UserID:           transaction.UserID,
		Amount:           transaction.Amount,
		Currency:         string(transaction.Currency),
		Balance:          transaction.Balance,
		OriginalAmount:   transaction.OriginalAmount,
		OriginalCurrency: transaction.OriginalCurrency,
		ConversionRate:   transaction.ConversionRate,
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
	return account.NewTransactionFromData(t.ID, t.UserID, t.AccountID, t.Amount, t.Balance, currency.Code(t.Currency), t.CreatedAt, t.OriginalAmount, t.OriginalCurrency, t.ConversionRate), nil
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
		tx = append(tx, account.NewTransactionFromData(t.ID, t.UserID, t.AccountID, t.Amount, t.Balance, currency.Code(t.Currency), t.CreatedAt, t.OriginalAmount, t.OriginalCurrency, t.ConversionRate))
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

func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{db: db}
}
