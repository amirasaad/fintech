// Package decorator provides decorator patterns for cross-cutting concerns in the application.
// It includes transaction management decorators that wrap business operations with
// automatic transaction handling, error recovery, and logging.
package decorator

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
)

// TransactionDecorator defines the interface for transaction management decorators.
// Now passes context and the UnitOfWork into the operation function for explicit repository access.
type TransactionDecorator interface {
	// Execute runs the provided operation within a transaction context, passing the UnitOfWork.
	// The operation function receives the UnitOfWork for repository access.
	Execute(ctx context.Context, operation func(uow repository.UnitOfWork) error) error
}

// UnitOfWorkTransactionDecorator implements TransactionDecorator for the Unit of Work pattern.
// Now passes context and the UnitOfWork into the operation function.
type UnitOfWorkTransactionDecorator struct {
	uowFactory func() (repository.UnitOfWork, error)
	logger     *slog.Logger
}

// NewUnitOfWorkTransactionDecorator creates a new UnitOfWorkTransactionDecorator instance.
//
// Parameters:
//   - uowFactory: A function that creates and returns a UnitOfWork instance. This function
//     should handle the creation of the unit of work and any associated resources.
//   - logger: A structured logger for recording transaction lifecycle events, errors,
//     and debugging information.
//
// Returns a configured TransactionDecorator that can be injected into services
// for automatic transaction management.
//
// Example:
//
//	uowFactory := func() (repository.UnitOfWork, error) {
//	    return infra.NewUnitOfWork(db)
//	}
//	transactionDecorator := decorator.NewUnitOfWorkTransactionDecorator(uowFactory, logger)
//
//	service := &AccountService{
//	    transaction: transactionDecorator,
//	}
func NewUnitOfWorkTransactionDecorator(
	uowFactory func() (repository.UnitOfWork, error),
	logger *slog.Logger,
) *UnitOfWorkTransactionDecorator {
	return &UnitOfWorkTransactionDecorator{
		uowFactory: uowFactory,
		logger:     logger,
	}
}

// Execute runs the operation within a transaction context using the Unit of Work pattern.
// It provides comprehensive transaction lifecycle management with automatic error handling
// and recovery mechanisms.
//
// Transaction Lifecycle:
// 1. Creates a new UnitOfWork using the factory function
// 2. Begins the transaction
// 3. Executes the provided operation function
// 4. Commits the transaction on success
// 5. Rolls back the transaction on any error or panic
//
// Error Handling:
// - UnitOfWork creation failures are logged and wrapped with descriptive errors
// - Transaction begin failures are logged and returned as errors
// - Operation failures trigger automatic rollback and return the original error
// - Commit failures trigger rollback and return a descriptive error
// - Panics are recovered, logged, and re-panicked after rollback
//
// Logging:
// - All transaction lifecycle events are logged with structured data
// - Errors include context information for debugging
// - Panic recovery includes panic value for analysis
//
// Parameters:
//   - operation: A function containing the business logic to execute within the transaction.
//     This function should return an error if the operation fails. The function should
//     not handle transaction management - that's handled by the decorator.
//
// Returns:
//   - nil if the operation completes successfully
//   - An error if any part of the transaction lifecycle fails
//
// Example:
//
//	err := decorator.Execute(func() error {
//	    // Business logic only
//	    account, err := account.New().WithUserID(userID).Build()
//	    if err != nil {
//	        return err
//	    }
//	    return repo.Create(account)
//	})
//	if err != nil {
//	    // Handle error - transaction was automatically rolled back
//	}
func (d *UnitOfWorkTransactionDecorator) Execute(ctx context.Context, operation func(uow repository.UnitOfWork) error) error {
	uow, err := d.uowFactory()
	if err != nil {
		d.logger.Error("Failed to create unit of work", "error", err)
		return errors.New("failed to create unit of work")
	}
	if err = uow.Begin(ctx); err != nil {
		d.logger.Error("Failed to begin transaction", "error", err)
		return errors.New("failed to begin transaction")
	}
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("Transaction panic recovered", "panic", r)
			_ = uow.Rollback(ctx) //nolint:errcheck
			panic(r)
		}
	}()
	if err = operation(uow); err != nil {
		if rbErr := uow.Rollback(ctx); rbErr != nil {
			d.logger.Error("Failed to rollback transaction", "error", rbErr)
		}
		d.logger.Error("Transaction operation failed", "error", err)
		return err
	}
	if err = uow.Commit(ctx); err != nil {
		if rbErr := uow.Rollback(ctx); rbErr != nil {
			d.logger.Error("Failed to rollback transaction after commit error", "error", rbErr)
		}
		d.logger.Error("Failed to commit transaction", "error", err)
		return errors.New("failed to commit transaction")
	}
	return nil
}
