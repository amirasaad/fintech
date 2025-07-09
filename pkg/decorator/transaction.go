// Package decorator provides decorator patterns for cross-cutting concerns in the application.
// It includes transaction management decorators that wrap business operations with
// automatic transaction handling, error recovery, and logging.
package decorator

import (
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
)

// TransactionDecorator defines the interface for transaction management decorators.
// It provides a clean abstraction for wrapping business operations with transaction
// lifecycle management, including automatic begin, commit, and rollback handling.
//
// The decorator pattern allows business logic to focus on domain operations while
// the decorator handles all transaction-related concerns like:
// - Transaction lifecycle management (begin/commit/rollback)
// - Error handling and automatic rollback on failures
// - Panic recovery with proper cleanup
// - Structured logging of transaction events
//
// Example usage:
//
//	type AccountService struct {
//	    transaction decorator.TransactionDecorator
//	}
//
//	func (s *AccountService) CreateAccount(userID uuid.UUID) (*account.Account, error) {
//	    var account *account.Account
//	    err := s.transaction.Execute(func() error {
//	        // Business logic only - no transaction boilerplate
//	        account = account.New().WithUserID(userID).Build()
//	        return repo.Create(account)
//	    })
//	    return account, err
//	}
type TransactionDecorator interface {
	// Execute runs the provided operation within a transaction context.
	// It automatically handles transaction lifecycle including:
	// - Beginning the transaction
	// - Executing the operation
	// - Committing on success or rolling back on error
	// - Panic recovery with rollback
	// - Structured logging of all events
	//
	// The operation function should contain only business logic and return
	// an error if the operation fails. The decorator will handle all
	// transaction management automatically.
	//
	// Returns an error if:
	// - Unit of Work creation fails
	// - Transaction begin fails
	// - The operation function returns an error
	// - Transaction commit fails
	// - A panic occurs during execution
	Execute(operation func() error) error
}

// UnitOfWorkTransactionDecorator implements TransactionDecorator for the Unit of Work pattern.
// It provides transaction management using a UnitOfWork factory function and includes
// comprehensive logging and error handling.
//
// This decorator is designed to work with the repository pattern and provides:
// - Automatic transaction lifecycle management
// - Panic recovery with proper cleanup
// - Structured logging for observability
// - Graceful error handling for all failure scenarios
//
// The decorator ensures that transactions are properly managed even in edge cases
// like panics, commit failures, or rollback failures.
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
func (d *UnitOfWorkTransactionDecorator) Execute(operation func() error) error {
	// Create UnitOfWork
	uow, err := d.uowFactory()
	if err != nil {
		d.logger.Error("Failed to create unit of work", "error", err)
		return errors.New("failed to create unit of work")
	}

	// Begin transaction
	if err = uow.Begin(); err != nil {
		d.logger.Error("Failed to begin transaction", "error", err)
		return errors.New("failed to begin transaction")
	}

	// Defer panic recovery and cleanup
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("Transaction panic recovered", "panic", r)
			_ = uow.Rollback() //nolint:errcheck
			panic(r)           // re-panic after rollback
		}
	}()

	// Execute the business operation
	if err = operation(); err != nil {
		// Rollback on operation failure
		if rbErr := uow.Rollback(); rbErr != nil {
			d.logger.Error("Failed to rollback transaction", "error", rbErr)
		}
		d.logger.Error("Transaction operation failed", "error", err)
		return err
	}

	// Commit transaction
	if err = uow.Commit(); err != nil {
		// Rollback when commit fails
		if rbErr := uow.Rollback(); rbErr != nil {
			d.logger.Error("Failed to rollback transaction after commit error", "error", rbErr)
		}
		d.logger.Error("Failed to commit transaction", "error", err)
		return errors.New("failed to commit transaction")
	}

	return nil
}
