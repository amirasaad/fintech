package decorator

import (
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
)

// TransactionDecorator defines the interface for transaction management
type TransactionDecorator interface {
	Execute(operation func() error) error
}

// UnitOfWorkTransactionDecorator implements TransactionDecorator for Unit of Work pattern
type UnitOfWorkTransactionDecorator struct {
	uowFactory func() (repository.UnitOfWork, error)
	logger     *slog.Logger
}

// NewUnitOfWorkTransactionDecorator creates a new transaction decorator
func NewUnitOfWorkTransactionDecorator(
	uowFactory func() (repository.UnitOfWork, error),
	logger *slog.Logger,
) *UnitOfWorkTransactionDecorator {
	return &UnitOfWorkTransactionDecorator{
		uowFactory: uowFactory,
		logger:     logger,
	}
}

// Execute runs the operation within a transaction
func (d *UnitOfWorkTransactionDecorator) Execute(operation func() error) error {
	uow, err := d.uowFactory()
	if err != nil {
		d.logger.Error("Failed to create unit of work", "error", err)
		return errors.New("failed to create unit of work")
	}

	if err = uow.Begin(); err != nil {
		d.logger.Error("Failed to begin transaction", "error", err)
		return errors.New("failed to begin transaction")
	}

	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("Transaction panic recovered", "panic", r)
			_ = uow.Rollback() //nolint:errcheck
			panic(r)           // re-panic after rollback
		}
	}()

	if err = operation(); err != nil {
		if rbErr := uow.Rollback(); rbErr != nil {
			d.logger.Error("Failed to rollback transaction", "error", rbErr)
		}
		d.logger.Error("Transaction operation failed", "error", err)
		return err
	}

	if err = uow.Commit(); err != nil {
		// Call Rollback when Commit fails, as expected by tests
		if rbErr := uow.Rollback(); rbErr != nil {
			d.logger.Error("Failed to rollback transaction after commit error", "error", rbErr)
		}
		d.logger.Error("Failed to commit transaction", "error", err)
		return errors.New("failed to commit transaction")
	}

	return nil
}
