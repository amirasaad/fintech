package repository

import "context"

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
