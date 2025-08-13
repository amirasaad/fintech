package repository

import (
	"context"
)

// UnitOfWork defines the contract for transactional work and type-safe
// repository access.
//
// Why is GetRepository part of UnitOfWork?
// - Ensures all repositories use the same DB session/transaction for true
// atomicity.
// - Keeps service code clean and focused on business logic.
// - Centralizes repository wiring and registry for maintainability.
// - Prevents accidental use of the wrong DB session (which would break
// transactionality).
// - Is idiomatic for Go UoW patterns and easy to mock in tests.
//
// Do runs the given function in a transaction boundary, providing a UnitOfWork
// for repository access.
// GetRepository provides type-safe access to repositories using the
// transaction session.
// Example usage:
//
//	repoAny, err := uow.GetRepository((*UserRepository)(nil))
//	repo := repoAny.(UserRepository)
type UnitOfWork interface {
	// Do executes the given function within a transaction boundary.
	// The provided function receives a UnitOfWork for repository access.
	// If the function returns an error, the transaction is rolled back.
	Do(ctx context.Context, fn func(uow UnitOfWork) error) error

	// GetRepository returns a repository of the requested type, bound to the
	// current transaction/session. This method is maintained for backward
	// compatibility but is deprecated in favor of type-safe methods.
	// Example:
	//   repoAny, err := uow.GetRepository((*UserRepository)(nil))
	//   repo := repoAny.(UserRepository)
	GetRepository(repoType any) (any, error)
}
