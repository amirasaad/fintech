package repository

import (
	"context"
	"fmt"

	"github.com/amirasaad/fintech/pkg/repository"
	"gorm.io/gorm"
)

// StringBasedUnitOfWork defines the contract for transactional work using string-based repository names.
type StringBasedUnitOfWork interface {
	// Do executes the given function within a transaction boundary.
	Do(ctx context.Context, fn func(uow StringBasedUnitOfWork) error) error

	// GetRepository returns a repository by name, bound to the current transaction/session.
	GetRepository(repoName string) (any, error)

	// Type-safe convenience methods
	AccountRepository() (repository.AccountRepository, error)
	TransactionRepository() (repository.TransactionRepository, error)
	UserRepository() (repository.UserRepository, error)
}

// Repository constants for type safety
const (
	AccountRepositoryName     = "account"
	TransactionRepositoryName = "transaction"
	UserRepositoryName        = "user"
)

// StringBasedUoW provides transaction boundary and repository access using string names.
type StringBasedUoW struct {
	db           *gorm.DB
	tx           *gorm.DB
	repoRegistry map[string]func(*gorm.DB) interface{}
}

// NewStringBasedUoW creates a new string-based UoW for the given *gorm.DB.
func NewStringBasedUoW(db *gorm.DB) *StringBasedUoW {
	return &StringBasedUoW{
		db: db,
		repoRegistry: map[string]func(*gorm.DB) interface{}{
			AccountRepositoryName:     func(db *gorm.DB) interface{} { return NewAccountRepository(db) },
			TransactionRepositoryName: func(db *gorm.DB) interface{} { return NewTransactionRepository(db) },
			UserRepositoryName:        func(db *gorm.DB) interface{} { return NewUserRepository(db) },
		},
	}
}

// Do runs the given function in a transaction boundary.
func (u *StringBasedUoW) Do(ctx context.Context, fn func(uow StringBasedUnitOfWork) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &StringBasedUoW{db: u.db, tx: tx, repoRegistry: u.repoRegistry}
		return fn(txnUow)
	})
}

// GetRepository provides access to repositories using string names.
func (u *StringBasedUoW) GetRepository(repoName string) (any, error) {
	constructor, ok := u.repoRegistry[repoName]
	if !ok {
		return nil, fmt.Errorf("unsupported repository name: %s", repoName)
	}
	repo := constructor(u.tx)
	return repo, nil
}

// Type-safe repository access methods

// AccountRepository returns the account repository bound to the current transaction
func (u *StringBasedUoW) AccountRepository() (repository.AccountRepository, error) {
	repoAny, err := u.GetRepository(AccountRepositoryName)
	if err != nil {
		return nil, err
	}
	return repoAny.(repository.AccountRepository), nil
}

// TransactionRepository returns the transaction repository bound to the current transaction
func (u *StringBasedUoW) TransactionRepository() (repository.TransactionRepository, error) {
	repoAny, err := u.GetRepository(TransactionRepositoryName)
	if err != nil {
		return nil, err
	}
	return repoAny.(repository.TransactionRepository), nil
}

// UserRepository returns the user repository bound to the current transaction
func (u *StringBasedUoW) UserRepository() (repository.UserRepository, error) {
	repoAny, err := u.GetRepository(UserRepositoryName)
	if err != nil {
		return nil, err
	}
	return repoAny.(repository.UserRepository), nil
}
