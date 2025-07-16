package account

import (
	"context"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/handler"
	"github.com/google/uuid"
)

// Chain provides a simplified interface for executing account operations
type Chain struct {
	chain *handler.AccountChain
}

// NewChain creates a new account chain with the given dependencies
func NewChain(deps config.Deps) *Chain {
	return &Chain{
		chain: handler.NewAccountChain(deps.Uow, deps.CurrencyConverter, deps.PaymentProvider, deps.Logger),
	}
}

// Deposit executes a deposit operation using the chain of responsibility pattern
func (c *Chain) Deposit(ctx context.Context, userID, accountID uuid.UUID, amount float64, currencyCode currency.Code, moneySource, paymentID string) (*handler.OperationResponse, error) {
	return c.chain.Deposit(ctx, userID, accountID, amount, currencyCode, moneySource, paymentID)
}

// Withdraw executes a withdraw operation to an external target using the chain of responsibility pattern
func (c *Chain) Withdraw(ctx context.Context, userID, accountID uuid.UUID, amount float64, currencyCode currency.Code, externalTarget handler.ExternalTarget, paymentID string) (*handler.OperationResponse, error) {
	// Add a new WithdrawExternal method to AccountChain to support this
	return c.chain.Withdraw(ctx, userID, accountID, amount, currencyCode, externalTarget, paymentID)
}

// Transfer executes a transfer operation using the chain of responsibility pattern
func (c *Chain) Transfer(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID uuid.UUID, amount float64, currencyCode currency.Code) (*handler.OperationResponse, error) {
	return c.chain.Transfer(ctx, senderUserID, receiverUserID, sourceAccountID, destAccountID, amount, currencyCode)
}

// OperationHandler defines the interface for handling account operations in the chain
type OperationHandler = handler.OperationHandler

// OperationRequest contains all the data needed for account operations
type OperationRequest = handler.OperationRequest

// OperationResponse contains the result of an account operation
type OperationResponse = handler.OperationResponse
