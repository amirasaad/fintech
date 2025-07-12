package repository

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/google/uuid"
)

// CleanGenericRepository provides type-safe CRUD operations without infrastructure coupling
type CleanGenericRepository[T any] interface {
	// Basic CRUD operations
	Create(ctx context.Context, entity *T) error
	Get(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	List(ctx context.Context) ([]*T, error)
	FindBy(ctx context.Context, query interface{}, args ...interface{}) ([]*T, error)
	FindOneBy(ctx context.Context, query interface{}, args ...interface{}) (*T, error)
}

// CleanUnitOfWork provides transaction management without infrastructure details
type CleanUnitOfWork interface {
	// Do executes a function within a transaction boundary
	Do(ctx context.Context, fn func(CleanUnitOfWork) error) error

	// Type-safe repository access methods
	AccountRepository() CleanGenericRepository[account.Account]
	TransactionRepository() CleanGenericRepository[account.Transaction]
	UserRepository() CleanGenericRepository[user.User]
}

// CleanUnitOfWorkImpl implements CleanUnitOfWork
type CleanUnitOfWorkImpl struct {
	accountRepo     CleanGenericRepository[account.Account]
	transactionRepo CleanGenericRepository[account.Transaction]
	userRepo        CleanGenericRepository[user.User]
	transactionMgr  TransactionManager
}

// TransactionManager abstracts transaction management
type TransactionManager interface {
	// ExecuteInTransaction runs a function within a transaction
	ExecuteInTransaction(ctx context.Context, fn func() error) error
}

// NewCleanUnitOfWork creates a new clean UOW instance
func NewCleanUnitOfWork(
	accountRepo CleanGenericRepository[account.Account],
	transactionRepo CleanGenericRepository[account.Transaction],
	userRepo CleanGenericRepository[user.User],
	txMgr TransactionManager,
) CleanUnitOfWork {
	return &CleanUnitOfWorkImpl{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		transactionMgr:  txMgr,
	}
}

// Do executes a function within a transaction
func (uow *CleanUnitOfWorkImpl) Do(ctx context.Context, fn func(CleanUnitOfWork) error) error {
	return uow.transactionMgr.ExecuteInTransaction(ctx, func() error {
		return fn(uow)
	})
}

// AccountRepository returns the account repository
func (uow *CleanUnitOfWorkImpl) AccountRepository() CleanGenericRepository[account.Account] {
	return uow.accountRepo
}

// TransactionRepository returns the transaction repository
func (uow *CleanUnitOfWorkImpl) TransactionRepository() CleanGenericRepository[account.Transaction] {
	return uow.transactionRepo
}

// UserRepository returns the user repository
func (uow *CleanUnitOfWorkImpl) UserRepository() CleanGenericRepository[user.User] {
	return uow.userRepo
}
