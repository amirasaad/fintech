package repository

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"gorm.io/gorm"
)

// CompleteGenericUnitOfWork provides type-safe repository access using generics
type CompleteGenericUnitOfWork interface {
	// Transaction management
	Do(ctx context.Context, fn func(CompleteGenericUnitOfWork) error) error

	// Type-safe repository access methods
	AccountRepository() GenericRepository[account.Account]
	TransactionRepository() GenericRepository[account.Transaction]
	UserRepository() GenericRepository[user.User]
}

// CompleteGenericUnitOfWorkImpl implements the complete generic UOW interface
type CompleteGenericUnitOfWorkImpl struct {
	db              *gorm.DB
	tx              *gorm.DB
	accountRepo     GenericRepository[account.Account]
	transactionRepo GenericRepository[account.Transaction]
	userRepo        GenericRepository[user.User]
}

// NewCompleteGenericUnitOfWork creates a new complete generic UOW instance
func NewCompleteGenericUnitOfWork(db *gorm.DB) CompleteGenericUnitOfWork {
	return &CompleteGenericUnitOfWorkImpl{
		db:              db,
		accountRepo:     NewGenericRepository[account.Account](db),
		transactionRepo: NewGenericRepository[account.Transaction](db),
		userRepo:        NewGenericRepository[user.User](db),
	}
}

// Do executes a function within a transaction
func (uow *CompleteGenericUnitOfWorkImpl) Do(ctx context.Context, fn func(CompleteGenericUnitOfWork) error) error {
	return uow.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &CompleteGenericUnitOfWorkImpl{
			db:              uow.db,
			tx:              tx,
			accountRepo:     NewGenericRepository[account.Account](tx),
			transactionRepo: NewGenericRepository[account.Transaction](tx),
			userRepo:        NewGenericRepository[user.User](tx),
		}
		return fn(txnUow)
	})
}

// AccountRepository returns the account repository
func (uow *CompleteGenericUnitOfWorkImpl) AccountRepository() GenericRepository[account.Account] {
	return uow.accountRepo
}

// TransactionRepository returns the transaction repository
func (uow *CompleteGenericUnitOfWorkImpl) TransactionRepository() GenericRepository[account.Transaction] {
	return uow.transactionRepo
}

// UserRepository returns the user repository
func (uow *CompleteGenericUnitOfWorkImpl) UserRepository() GenericRepository[user.User] {
	return uow.userRepo
}
