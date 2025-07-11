package repository

import "context"

// UnitOfWork defines the transaction boundary and repository provider for a business operation.
//
// Migration: Use GetRepository[T any]() for repository access. The old AccountRepository, TransactionRepository, and UserRepository methods are deprecated.
type UnitOfWork interface {
	Begin(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	// Generic repository accessor. Example: repo, err := uow.GetRepository[UserRepository]()
	GetRepository[T any]() (T, error)
	// Deprecated: Use GetRepository[T any]() instead.
	AccountRepository() (AccountRepository, error)
	TransactionRepository() (TransactionRepository, error)
	UserRepository() (UserRepository, error)
}
