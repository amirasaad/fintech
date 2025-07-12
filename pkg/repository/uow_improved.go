package repository

import (
	"context"
)

// ImprovedUnitOfWork provides type-safe repository access without reflect
type ImprovedUnitOfWork interface {
	// Transaction management
	Do(ctx context.Context, fn func(ImprovedUnitOfWork) error) error

	// Type-safe repository access methods
	AccountRepository() AccountRepository
	TransactionRepository() TransactionRepository
	UserRepository() UserRepository
}

// ImprovedUnitOfWorkImpl implements the improved UOW interface
type ImprovedUnitOfWorkImpl struct {
	accountRepo     AccountRepository
	transactionRepo TransactionRepository
	userRepo        UserRepository
}

// NewImprovedUnitOfWork creates a new improved UOW instance
func NewImprovedUnitOfWork(
	accountRepo AccountRepository,
	transactionRepo TransactionRepository,
	userRepo UserRepository,
) ImprovedUnitOfWork {
	return &ImprovedUnitOfWorkImpl{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
	}
}

// Do executes a function within a transaction
func (uow *ImprovedUnitOfWorkImpl) Do(ctx context.Context, fn func(ImprovedUnitOfWork) error) error {
	// Transaction logic here - this would be implemented by the concrete UOW
	return fn(uow)
}

// AccountRepository returns the account repository
func (uow *ImprovedUnitOfWorkImpl) AccountRepository() AccountRepository {
	return uow.accountRepo
}

// TransactionRepository returns the transaction repository
func (uow *ImprovedUnitOfWorkImpl) TransactionRepository() TransactionRepository {
	return uow.transactionRepo
}

// UserRepository returns the user repository
func (uow *ImprovedUnitOfWorkImpl) UserRepository() UserRepository {
	return uow.userRepo
}
