package repository

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/google/uuid"
)

// AccountRepository defines the interface for account data access operations.
type AccountRepository interface {
	Get(id uuid.UUID) (*account.Account, error)
	Create(account *account.Account) error
	Update(account *account.Account) error
	Delete(id uuid.UUID) error
}

// TransactionRepository defines the interface for transaction data access operations.
type TransactionRepository interface {
	Create(transaction *account.Transaction, convInfo *common.ConversionInfo,
		maskedExternalTarget string) error
	Get(id uuid.UUID) (*account.Transaction, error)
	List(userID, accountID uuid.UUID) ([]*account.Transaction, error)
	// GetByPaymentID returns a transaction by its payment provider ID.
	GetByPaymentID(paymentID string) (*account.Transaction, error)
	Update(tx *account.Transaction) error
}

// UserRepository defines the interface for user data access operations.
type UserRepository interface {
	Get(id uuid.UUID) (*user.User, error)
	GetByUsername(username string) (*user.User, error)
	GetByEmail(email string) (*user.User, error)
	Valid(id uuid.UUID, password string) bool
	Create(user *user.User) error
	Update(user *user.User) error
	Delete(id uuid.UUID) error
}
