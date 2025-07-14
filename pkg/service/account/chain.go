package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/handler"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// Chain provides a simplified interface for executing account operations
type Chain struct {
	chain *handler.AccountChain
}

// NewChain creates a new account chain with the given dependencies
func NewChain(uow repository.UnitOfWork, converter mon.CurrencyConverter, logger *slog.Logger) *Chain {
	return &Chain{
		chain: handler.NewAccountChain(uow, converter, logger),
	}
}

// Deposit executes a deposit operation using the chain of responsibility pattern
func (c *Chain) Deposit(ctx context.Context, userID, accountID uuid.UUID, amount float64, currencyCode currency.Code, moneySource string) (*handler.OperationResponse, error) {
	return c.chain.Deposit(ctx, userID, accountID, amount, currencyCode, moneySource)
}

// Withdraw executes a withdraw operation using the chain of responsibility pattern
func (c *Chain) Withdraw(ctx context.Context, userID, accountID uuid.UUID, amount float64, currencyCode currency.Code, moneySource string) (*handler.OperationResponse, error) {
	return c.chain.Withdraw(ctx, userID, accountID, amount, currencyCode, moneySource)
}

// Transfer executes a transfer operation using the chain of responsibility pattern
func (c *Chain) Transfer(ctx context.Context, userID, sourceAccountID, destAccountID uuid.UUID, amount float64, currencyCode currency.Code, moneySource string) (*handler.OperationResponse, error) {
	return c.chain.Transfer(ctx, userID, sourceAccountID, destAccountID, amount, currencyCode, moneySource)
}

// OperationHandler defines the interface for handling account operations in the chain
type OperationHandler = handler.OperationHandler

// OperationRequest contains all the data needed for account operations
type OperationRequest = handler.OperationRequest

// OperationResponse contains the result of an account operation
type OperationResponse = handler.OperationResponse
