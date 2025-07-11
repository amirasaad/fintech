package repository

import (
	"context"
	"fmt"
	"reflect"

	"github.com/amirasaad/fintech/pkg/repository"
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
	db           *gorm.DB
	tx           *gorm.DB
	repoRegistry map[reflect.Type]func(*gorm.DB) interface{}
}

// NewUoW creates a new UoW for the given *gorm.DB.
func NewUoW(db *gorm.DB) *UoW {
	return &UoW{
		db: db,
		repoRegistry: map[reflect.Type]func(*gorm.DB) interface{}{
			reflect.TypeOf((*repository.AccountRepository)(nil)).Elem():     func(db *gorm.DB) interface{} { return NewAccountRepository(db) },
			reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewTransactionRepository(db) },
			reflect.TypeOf((*repository.UserRepository)(nil)).Elem():        func(db *gorm.DB) interface{} { return NewUserRepository(db) },
		},
	}
}

// Do runs the given function in a transaction boundary, providing a UoW with repository access.
func (u *UoW) Do(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &UoW{db: u.db, tx: tx, repoRegistry: u.repoRegistry}
		return fn(txnUow)
	})
}

// GetRepository provides generic, type-safe access to repositories using the transaction session.
//
// This method is part of UoW to guarantee that all repository operations within a transaction
// use the same DB session, ensuring atomicity and consistency. It also centralizes repository
// construction and makes testing and extension easier.
func (u *UoW) GetRepository(repoType reflect.Type) (any, error) {
	constructor, ok := u.repoRegistry[repoType]
	if !ok {
		return nil, fmt.Errorf("unsupported repository type: %v", repoType)
	}
	repo := constructor(u.tx)
	return repo, nil
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
// func (m *MockUnitOfWork) GetRepository(repoType reflect.Type) (any, error) {
//     if m.GetRepositoryFunc != nil {
//         return m.GetRepositoryFunc(repoType)
//     }
//     return nil, nil
// }
