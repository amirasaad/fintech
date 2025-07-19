package repository

import (
	"context"
	"fmt"

	repoaccount "github.com/amirasaad/fintech/infra/repository/account"
	repotransaction "github.com/amirasaad/fintech/infra/repository/transaction"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"gorm.io/gorm"
)

// UoW provides transaction boundary and repository access in one abstraction.
//
// Why is GetRepository part of UoW?
// - Ensures all repositories use the same DB session/transaction for true atomicity.
// - Keeps service code clean and focused on business logic.
// - Centralizes repository wiring and registry for maintainability.
// - Prevents accidental use of the wrong DB session (which would break transactionality).
// - Is idiomatic for Go UoW patterns and easy to mock in tests.
type UoW struct {
	db *gorm.DB
	tx *gorm.DB

	// Direct repository instances for type-safe access
	accountRepo     repository.AccountRepository
	transactionRepo repository.TransactionRepository
	userRepo        repository.UserRepository
}

// NewUoW creates a new UoW for the given *gorm.DB.
func NewUoW(db *gorm.DB) *UoW {
	return &UoW{
		db: db,
	}
}

// Do runs the given function in a transaction boundary, providing a UoW with repository access.
func (u *UoW) Do(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &UoW{
			db:              u.db,
			tx:              tx,
			accountRepo:     NewAccountRepository(tx),
			transactionRepo: NewTransactionRepository(tx),
			userRepo:        NewUserRepository(tx),
		}
		return fn(txnUow)
	})
}

// GetRepository provides generic, type-safe access to repositories using the transaction session.
// This method is maintained for backward compatibility but is deprecated in favor of type-safe methods.
//
// This method is part of UoW to guarantee that all repository operations within a transaction
// use the same DB session, ensuring atomicity and consistency. It also centralizes repository
// construction and makes testing and extension easier.
func (u *UoW) GetRepository(repoType interface{}) (any, error) {
	// Use transaction DB if available, otherwise use main DB
	dbToUse := u.tx
	if dbToUse == nil {
		dbToUse = u.db
	}

	// Create repositories on-demand for backward compatibility and CQRS
	switch repoType {
	case (*repository.AccountRepository)(nil):
		return NewAccountRepository(dbToUse), nil
	case (*repository.TransactionRepository)(nil):
		return NewTransactionRepository(dbToUse), nil
	case (*repository.UserRepository)(nil):
		return NewUserRepository(dbToUse), nil
	// --- CQRS-style repositories ---
	case (*account.Repository)(nil):
		return repoaccount.New(dbToUse), nil
	case (*transaction.Repository)(nil):
		return repotransaction.New(dbToUse), nil
	default:
		return nil, fmt.Errorf("unsupported repository type: %T", repoType)
	}
}

// Type-safe repository access methods (preferred approach)

// AccountRepository returns the account repository bound to the current transaction
func (u *UoW) AccountRepository() (repository.AccountRepository, error) {
	if u.accountRepo != nil {
		return u.accountRepo, nil
	}

	// Create repository on-demand if not already created
	dbToUse := u.tx
	if dbToUse == nil {
		dbToUse = u.db
	}

	u.accountRepo = NewAccountRepository(dbToUse)
	return u.accountRepo, nil
}

// TransactionRepository returns the transaction repository bound to the current transaction
func (u *UoW) TransactionRepository() (repository.TransactionRepository, error) {
	if u.transactionRepo != nil {
		return u.transactionRepo, nil
	}

	// Create repository on-demand if not already created
	dbToUse := u.tx
	if dbToUse == nil {
		dbToUse = u.db
	}

	u.transactionRepo = NewTransactionRepository(dbToUse)
	return u.transactionRepo, nil
}

// UserRepository returns the user repository bound to the current transaction
func (u *UoW) UserRepository() (repository.UserRepository, error) {
	if u.userRepo != nil {
		return u.userRepo, nil
	}

	// Create repository on-demand if not already created
	dbToUse := u.tx
	if dbToUse == nil {
		dbToUse = u.db
	}

	u.userRepo = NewUserRepository(dbToUse)
	return u.userRepo, nil
}

// ---
// Sample mock for tests:
//
// type MockUnitOfWork struct {
//     DoFunc func(ctx context.Context, fn func(uow UnitOfWork) error) error
//     GetRepositoryFunc func(repoType any) (any, error)
// }
//
// func (m *MockUnitOfWork) Do(ctx context.Context, fn func(uow UnitOfWork) error) error {
//     if m.DoFunc != nil { return m.DoFunc(ctx, fn) }
//     return fn(m)
// }
// func (m *MockUnitOfWork) GetRepository(repoType any) (any, error) {
//     if m.GetRepositoryFunc != nil {
//         return m.GetRepositoryFunc(repoType)
//     }
//     return nil, nil
// }
