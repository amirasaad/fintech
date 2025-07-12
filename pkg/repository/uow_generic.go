package repository

import (
	"context"
	"fmt"
)

// GenericUnitOfWork provides type-safe repository access using generics
type GenericUnitOfWork interface {
	// Transaction management
	Do(ctx context.Context, fn func(GenericUnitOfWork) error) error
}

// GenericUnitOfWorkImpl implements the generic UOW interface
type GenericUnitOfWorkImpl struct {
	repositories map[string]any
}

// NewGenericUnitOfWork creates a new generic UOW instance
func NewGenericUnitOfWork() *GenericUnitOfWorkImpl {
	return &GenericUnitOfWorkImpl{
		repositories: make(map[string]any),
	}
}

// RegisterRepository registers a repository with a type key
func (uow *GenericUnitOfWorkImpl) RegisterRepository(repo any) {
	typeName := fmt.Sprintf("%T", repo)
	uow.repositories[typeName] = repo
}

// GetRepository retrieves a repository by type
func (uow *GenericUnitOfWorkImpl) GetRepository(repoType interface{}) (interface{}, error) {
	typeName := fmt.Sprintf("%T", repoType)

	repo, exists := uow.repositories[typeName]
	if !exists {
		return nil, fmt.Errorf("repository not found for type: %s", typeName)
	}

	return repo, nil
}

// Do executes a function within a transaction
func (uow *GenericUnitOfWorkImpl) Do(ctx context.Context, fn func(GenericUnitOfWork) error) error {
	// Transaction logic here
	return fn(uow)
}

// Type-safe helper methods for common repositories
func (uow *GenericUnitOfWorkImpl) GetAccountRepository() (AccountRepository, error) {
	repo, err := uow.GetRepository((*AccountRepository)(nil))
	if err != nil {
		return nil, err
	}
	if accountRepo, ok := repo.(AccountRepository); ok {
		return accountRepo, nil
	}
	return nil, fmt.Errorf("repository is not AccountRepository")
}

func (uow *GenericUnitOfWorkImpl) GetTransactionRepository() (TransactionRepository, error) {
	repo, err := uow.GetRepository((*TransactionRepository)(nil))
	if err != nil {
		return nil, err
	}
	if txRepo, ok := repo.(TransactionRepository); ok {
		return txRepo, nil
	}
	return nil, fmt.Errorf("repository is not TransactionRepository")
}

func (uow *GenericUnitOfWorkImpl) GetUserRepository() (UserRepository, error) {
	repo, err := uow.GetRepository((*UserRepository)(nil))
	if err != nil {
		return nil, err
	}
	if userRepo, ok := repo.(UserRepository); ok {
		return userRepo, nil
	}
	return nil, fmt.Errorf("repository is not UserRepository")
}
