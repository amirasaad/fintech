package repository

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

// UnitOfWork defines the contract for transactional work and type-safe repository access.
//
// Do runs the given function in a transaction boundary, providing a UnitOfWork for repository access.
// GetRepository provides generic, type-safe access to repositories using the transaction session.
type UnitOfWork interface {
	// Do executes the given function within a transaction boundary.
	// The provided function receives a UnitOfWork for repository access.
	// If the function returns an error, the transaction is rolled back.
	Do(ctx context.Context, fn func(uow UnitOfWork) error) error

	// GetRepository returns a repository of the requested type, bound to the current transaction/session.
	// Example: repo, err := uow.GetRepository[UserRepository]()
	GetRepository[T any]() (T, error)
}

// UoW provides transaction boundary and repository access in one abstraction.
// Usage:
//   uow := NewUoW(db)
//   err := uow.Do(ctx, func(uow UnitOfWork) error {
//       repo, err := uow.GetRepository[repository.UserRepository]()
//       ...
//   })
type UoW struct {
	db          *gorm.DB
	tx          *gorm.DB
	repoRegistry map[reflect.Type]func(*gorm.DB) interface{}
}

// NewUoW creates a new UoW for the given *gorm.DB.
func NewUoW(db *gorm.DB) *UoW {
	return &UoW{
		db: db,
		repoRegistry: map[reflect.Type]func(*gorm.DB) interface{}{
			reflect.TypeOf((*AccountRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewAccountRepository(db) },
			reflect.TypeOf((*TransactionRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewTransactionRepository(db) },
			reflect.TypeOf((*UserRepository)(nil)).Elem(): func(db *gorm.DB) interface{} { return NewUserRepository(db) },
		},
	}
}

// Do runs the given function in a transaction boundary, providing a UoW with repository access.
func (u *UoW) Do(ctx context.Context, fn func(uow UnitOfWork) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txnUow := &UoW{db: u.db, tx: tx, repoRegistry: u.repoRegistry}
		return fn(txnUow)
	})
}

// GetRepository provides generic, type-safe access to repositories using the transaction session.
func (u *UoW) GetRepository[T any]() (T, error) {
	var zero T
	t := reflect.TypeOf((*T)(nil)).Elem()
	constructor, ok := u.repoRegistry[t]
	if !ok {
		return zero, fmt.Errorf("unsupported repository type: %v", t)
	}
	repo := constructor(u.tx)
	return repo.(T), nil
}

// ---
// Sample mock for tests:
//
// type MockUnitOfWork struct {
//     DoFunc func(ctx context.Context, fn func(uow UnitOfWork) error) error
//     GetRepositoryFunc func(any) (any, error)
// }
//
// func (m *MockUnitOfWork) Do(ctx context.Context, fn func(uow UnitOfWork) error) error {
//     if m.DoFunc != nil { return m.DoFunc(ctx, fn) }
//     return fn(m)
// }
// func (m *MockUnitOfWork) GetRepository[T any]() (T, error) {
//     if m.GetRepositoryFunc != nil {
//         repo, err := m.GetRepositoryFunc(new(T))
//         return repo.(T), err
//     }
//     var zero T
//     return zero, nil
// }
