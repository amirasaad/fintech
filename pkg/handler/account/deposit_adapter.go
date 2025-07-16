package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/repository"
)

type DepositAdapter struct {
	uow    repository.UnitOfWork
	logger *slog.Logger
}

func NewDepositAdapter(uow repository.UnitOfWork, logger *slog.Logger) *DepositAdapter {
	return &DepositAdapter{uow: uow, logger: logger}
}

// Implement the DepositService interface methods here as needed.
// For example:
// func (a *DepositAdapter) Deposit(ctx context.Context, userID, accountID string, amount float64, currency string) error {
// 	// TODO: Add domain logic
// 	return nil
// }

// Add Deposit method to DepositAdapter to implement the expected interface
func (a *DepositAdapter) Deposit(ctx context.Context, userID, accountID string, amount float64, currency string) error {
	// TODO: Add domain logic
	return nil
}
