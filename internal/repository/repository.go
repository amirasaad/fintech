package repository

import (
	"github.com/amirasaad/fintech/internal/domain"
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
