package repository

import (
	"context"

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
	Create(transaction *account.Transaction, convInfo *common.ConversionInfo, maskedExternalTarget string) error
	Get(id uuid.UUID) (*account.Transaction, error)
	List(userID, accountID uuid.UUID) ([]*account.Transaction, error)
	// GetByPaymentID returns a transaction by its payment provider ID.
	GetByPaymentID(paymentID string) (*account.Transaction, error)
	Update(tx *account.Transaction) error
}

// TransactionUpdate is used for updating one or more fields of a transaction.
type TransactionUpdate struct {
	Status    *account.TransactionStatus
	PaymentID *string
	// Add more fields as needed for future updates
}

// TransactionRepositoryV2 defines a flexible interface for transaction data access operations with support for partial updates, upserts, and business-key-based queries.
type TransactionRepositoryV2 interface {
	// Create inserts a new transaction record.
	Create(ctx context.Context, transaction *account.Transaction) error

	// Update updates an existing transaction by its ID.
	Update(ctx context.Context, transaction *account.Transaction) error

	// PartialUpdate updates specified fields of a transaction by its ID.
	PartialUpdate(ctx context.Context, id uuid.UUID, update TransactionUpdate) error

	// Upsert inserts or updates a transaction by a business key (e.g., event ID, payment ID).
	UpsertByPaymentID(ctx context.Context, paymentID string, transaction *account.Transaction) error

	// Get retrieves a transaction by its ID.
	Get(ctx context.Context, id uuid.UUID) (*account.Transaction, error)

	// GetByPaymentID retrieves a transaction by its payment provider ID.
	GetByPaymentID(ctx context.Context, paymentID string) (*account.Transaction, error)

	// ListByUser lists all transactions for a given user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*account.Transaction, error)

	// ListByAccount lists all transactions for a given account.
	ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*account.Transaction, error)
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
