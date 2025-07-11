// Package decorator provides decorator patterns for cross-cutting concerns in the application.
// It includes transaction management decorators that wrap business operations with
// automatic transaction handling, error recovery, and logging.
package decorator

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
	"gorm.io/gorm"
)

// TransactionDecorator defines the interface for transaction management decorators.
// Now passes context and the UnitOfWork into the operation function for explicit repository access.
type TransactionDecorator interface {
	// Execute runs the provided operation within a transaction context, passing the UnitOfWork.
	// The operation function receives the UnitOfWork for repository access.
	Execute(ctx context.Context, operation func(uow repository.UnitOfWork) error) error
}

// UnitOfWorkTransactionDecorator implements TransactionDecorator for the Unit of Work pattern.
// Now uses GORM's Transaction method for safe transaction management.
type UnitOfWorkTransactionDecorator struct {
	dbFactory func() (*gorm.DB, error) // returns a *gorm.DB session
	logger   *slog.Logger
}

// NewUnitOfWorkTransactionDecorator creates a new decorator using a dbFactory and logger.
func NewUnitOfWorkTransactionDecorator(
	dbFactory func() (*gorm.DB, error),
	logger *slog.Logger,
) *UnitOfWorkTransactionDecorator {
	return &UnitOfWorkTransactionDecorator{
		dbFactory: dbFactory,
		logger:   logger,
	}
}

// Execute runs the operation in a GORM transaction, passing a new UoW for the transaction session.
func (d *UnitOfWorkTransactionDecorator) Execute(ctx context.Context, operation func(uow repository.UnitOfWork) error) error {
	db, err := d.dbFactory()
	if err != nil {
		d.logger.Error("Failed to get DB from factory", "error", err)
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		uow := repository.NewGormUoW(tx)
		return operation(uow)
	})
}
