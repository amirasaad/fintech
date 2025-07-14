package repository

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/google/uuid"
)

// AccountRepository defines the interface for account data access operations.
type AccountRepository interface {
	Get(id uuid.UUID) (*domain.Account, error)
	Create(account *domain.Account) error
	Update(account *domain.Account) error
	Delete(id uuid.UUID) error
}

// TransactionRepository defines the interface for transaction data access operations.
type TransactionRepository interface {
	Create(transaction *domain.Transaction, convInfo *common.ConversionInfo, externalTargetMasked string) error
	Get(id uuid.UUID) (*domain.Transaction, error)
	List(userID, accountID uuid.UUID) ([]*domain.Transaction, error)
	// GetByPaymentID returns a transaction by its payment provider ID.
	GetByPaymentID(paymentID string) (*domain.Transaction, error)
	Update(tx *domain.Transaction) error
}

// UserRepository defines the interface for user data access operations.
type UserRepository interface {
	Get(id uuid.UUID) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
	Valid(id uuid.UUID, password string) bool
	Create(user *domain.User) error
	Update(user *domain.User) error
	Delete(id uuid.UUID) error
}
