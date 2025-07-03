package infra

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
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
	var a domain.Account
	result := r.db.First(&a, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return domain.NewAccountFromData(a.ID, uuid.New(), a.Balance, a.CreatedAt, a.UpdatedAt), nil
}

func (r *accountRepository) Create(a *domain.Account) error {
	result := r.db.Create(a)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *accountRepository) Update(a *domain.Account) error {
	dbModel := domain.Account{
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
	return domain.NewTransactionFromData(t.ID, t.AccountID, t.Amount, t.Balance, t.CreatedAt), nil
}

func (r *transactionRepository) List(userID, accountID uuid.UUID) ([]*domain.Transaction, error) {
	var dbTransactions []*Transaction
	result := r.db.Where("account_id = ? and user_id = ?", accountID, userID).Order("created desc").Limit(100).Find(&dbTransactions)
	if result.Error != nil {
		return nil, result.Error
	}
	tx := make([]*domain.Transaction, 0, len(dbTransactions))
	for _, t := range dbTransactions {
		tx = append(tx, domain.NewTransactionFromData(t.ID, t.AccountID, t.Amount, t.Balance, t.CreatedAt))
	}
	return tx, nil
}

type userRepository struct {
	db *gorm.DB
}

// Valid implements repository.UserRepository.
func (u *userRepository) Valid(id uuid.UUID, password string) bool {
	var user domain.User
	result := u.db.Where("id = ? AND password = ?", id, password).First(&user)
	return result.Error == nil
}

// Create implements repository.UserRepository.
func (u *userRepository) Create(user *domain.User) error {
	result := u.db.Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Delete implements repository.UserRepository.
func (u *userRepository) Delete(id uuid.UUID) error {
	result := u.db.Delete(&domain.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Get implements repository.UserRepository.
func (u *userRepository) Get(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	result := u.db.First(&user, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetByEmail implements repository.UserRepository.
func (u *userRepository) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := u.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetByUsername implements repository.UserRepository.
func (u *userRepository) GetByUsername(username string) (*domain.User, error) {
	var user domain.User
	result := u.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// Update implements repository.UserRepository.
func (u *userRepository) Update(user *domain.User) error {
	result := u.db.Save(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{db: db}
}
