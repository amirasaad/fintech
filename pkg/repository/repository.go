package repository

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
)

type AccountRepository interface {
	Get(id uuid.UUID) (*domain.Account, error)
	Create(account *domain.Account) error
	Update(account *domain.Account) error
	Delete(id uuid.UUID) error
}

type TransactionRepository interface {
	Create(transaction *domain.Transaction) error
	Get(id uuid.UUID) (*domain.Transaction, error)
	List(accountID uuid.UUID) ([]*domain.Transaction, error)
}

type UserRepository interface {
	Get(id uuid.UUID) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
	Valid(id uuid.UUID, password string) bool
	Create(user *domain.User) error
	Update(user *domain.User) error
	Delete(id uuid.UUID) error
}
