package repository

import (
	"context"
)

// UnitOfWork defines the contract for transactional work and type-safe repository access.
//
// Why is GetRepository part of UnitOfWork?
// - Ensures all repositories use the same DB session/transaction for true atomicity.
// - Keeps service code clean and focused on business logic.
// - Centralizes repository wiring and registry for maintainability.
// - Prevents accidental use of the wrong DB session (which would break transactionality).
// - Is idiomatic for Go UoW patterns and easy to mock in tests.
//
// Do runs the given function in a transaction boundary, providing a UnitOfWork for repository access.
// GetRepository provides type-safe access to repositories using the transaction session.
// DEPRECATED: Use AccountRepository() or TransactionRepository() for all new code. This method is maintained for backward compatibility only.
//
// This method is part of UoW to guarantee that all repository operations within a transaction
// use the same DB session, ensuring atomicity and consistency. It also centralizes repository
// construction and makes testing and extension easier.
func (u *UoW) GetRepository(repoType interface{}) (any, error) {
	// ... existing code ...
}

// Type-safe repository access methods (preferred approach)
func (u *UoW) AccountRepository() (AccountRepository, error) {
	// ... existing code ...
}
func (u *UoW) TransactionRepository() (TransactionRepository, error) {
	// ... existing code ...
}
func (u *UoW) UserRepository() (UserRepository, error) {
	// ... existing code ...
}
