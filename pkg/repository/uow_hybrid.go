package repository

import (
	"context"
	"fmt"
	"reflect"
)

// HybridUnitOfWork combines the current reflect-based approach with type-safe alternatives
type HybridUnitOfWork interface {
	// Current interface (for backward compatibility)
	Do(ctx context.Context, fn func(uow HybridUnitOfWork) error) error
	GetRepository(repoType reflect.Type) (any, error)

	// Type-safe alternatives (preferred)
	AccountRepository() AccountRepository
	TransactionRepository() TransactionRepository
	UserRepository() UserRepository
}

// HybridUnitOfWorkImpl implements the hybrid UOW interface
type HybridUnitOfWorkImpl struct {
	// For backward compatibility
	repoRegistry map[reflect.Type]func() interface{}

	// Direct repository access
	accountRepo     AccountRepository
	transactionRepo TransactionRepository
	userRepo        UserRepository
}

// NewHybridUnitOfWork creates a new hybrid UOW instance
func NewHybridUnitOfWork(
	accountRepo AccountRepository,
	transactionRepo TransactionRepository,
	userRepo UserRepository,
) HybridUnitOfWork {
	return &HybridUnitOfWorkImpl{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		repoRegistry: map[reflect.Type]func() interface{}{
			reflect.TypeOf((*AccountRepository)(nil)).Elem():     func() interface{} { return accountRepo },
			reflect.TypeOf((*TransactionRepository)(nil)).Elem(): func() interface{} { return transactionRepo },
			reflect.TypeOf((*UserRepository)(nil)).Elem():        func() interface{} { return userRepo },
		},
	}
}

// Do executes a function within a transaction
func (uow *HybridUnitOfWorkImpl) Do(ctx context.Context, fn func(HybridUnitOfWork) error) error {
	// Transaction logic here
	return fn(uow)
}

// GetRepository provides backward compatibility with reflect-based access
func (uow *HybridUnitOfWorkImpl) GetRepository(repoType reflect.Type) (any, error) {
	constructor, ok := uow.repoRegistry[repoType]
	if !ok {
		return nil, fmt.Errorf("unsupported repository type: %v", repoType)
	}
	return constructor(), nil
}

// Type-safe repository access methods (preferred)
func (uow *HybridUnitOfWorkImpl) AccountRepository() AccountRepository {
	return uow.accountRepo
}

func (uow *HybridUnitOfWorkImpl) TransactionRepository() TransactionRepository {
	return uow.transactionRepo
}

func (uow *HybridUnitOfWorkImpl) UserRepository() UserRepository {
	return uow.userRepo
}
