package handler

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// AccountChain provides a simplified interface for executing account operations
type AccountChain struct {
	builder *ChainBuilder
}

// NewAccountChain creates a new account chain with the given dependencies
func NewAccountChain(uow repository.UnitOfWork, converter money.CurrencyConverter, provider provider.PaymentProvider, logger *slog.Logger) *AccountChain {
	return &AccountChain{
		builder: NewChainBuilder(uow, converter, provider, logger),
	}
}

// Deposit executes a deposit operation using the chain of responsibility pattern
func (c *AccountChain) Deposit(ctx context.Context, userID, accountID uuid.UUID, amount float64, currencyCode currency.Code, moneySource, paymentID string) (*OperationResponse, error) {
	chain := c.builder.BuildDepositChain()

	req := &OperationRequest{
		UserID:       userID,
		AccountID:    accountID,
		PaymentID:    paymentID,
		Amount:       amount,
		CurrencyCode: currencyCode,
		Operation:    OperationDeposit,
		MoneySource:  moneySource,
	}

	return chain.Handle(ctx, req)
}

// Withdraw executes a withdraw operation to an external target using the chain of responsibility pattern
func (c *AccountChain) Withdraw(ctx context.Context, userID, accountID uuid.UUID, amount float64, currencyCode currency.Code, externalTarget ExternalTarget, paymentID string) (*OperationResponse, error) {
	chain := c.builder.BuildWithdrawChain()

	req := &OperationRequest{
		UserID:         userID,
		AccountID:      accountID,
		PaymentID:      paymentID,
		Amount:         amount,
		CurrencyCode:   currencyCode,
		Operation:      OperationWithdraw,
		MoneySource:    "External",
		ExternalTarget: &externalTarget,
	}

	return chain.Handle(ctx, req)
}

// Transfer executes a transfer operation using the chain of responsibility pattern
func (c *AccountChain) Transfer(ctx context.Context, senderUserID, receiverUserID, sourceAccountID, destAccountID uuid.UUID, amount float64, currencyCode currency.Code) (*OperationResponse, error) {
	chain := c.builder.BuildTransferChain()

	req := &OperationRequest{
		UserID:        senderUserID,
		DestUserID:    receiverUserID,
		AccountID:     sourceAccountID,
		DestAccountID: destAccountID,
		Amount:        amount,
		CurrencyCode:  currencyCode,
		Operation:     OperationTransfer,
		MoneySource:   string(account.MoneySourceInternal),
	}

	return chain.Handle(ctx, req)
}
