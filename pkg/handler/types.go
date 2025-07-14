package handler

import (
	"context"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// OperationType represents the type of account operation
type OperationType string

// Operation type constants define the types of account operations.
const (
	OperationDeposit  OperationType = "deposit"
	OperationWithdraw OperationType = "withdraw"
	OperationTransfer OperationType = "transfer"
)

// OperationHandler defines the interface for handling account operations in the chain
type OperationHandler interface {
	Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error)
	SetNext(handler OperationHandler)
}

// OperationRequest contains all the data needed for account operations
type OperationRequest struct {
	UserID         uuid.UUID
	AccountID      uuid.UUID
	Amount         float64
	CurrencyCode   currency.Code
	Operation      OperationType
	Account        *account.Account
	Money          mon.Money
	ConvertedMoney mon.Money
	ConvInfo       *common.ConversionInfo
	ConvInfoOut    *common.ConversionInfo // Outgoing (source) conversion info for transfer
	ConvInfoIn     *common.ConversionInfo // Incoming (dest) conversion info for transfer
	Transaction    *account.Transaction
	TransactionIn  *account.Transaction // For transfer operations
	// For transfer
	DestAccountID uuid.UUID
	DestAccount   *account.Account
	DestUserID    uuid.UUID
	MoneySource   string // Origin of funds for deposit (e.g., Cash, Stripe, etc.)
}

// OperationResponse contains the result of an account operation
// For transfers, both TransactionOut (from source) and TransactionIn (to dest) may be set.
type OperationResponse struct {
	Transaction    *account.Transaction // For single-op or outgoing transfer
	TransactionOut *account.Transaction // Outgoing (source) transaction for transfer
	TransactionIn  *account.Transaction // Incoming (dest) transaction for transfer
	ConvInfo       *common.ConversionInfo
	ConvInfoOut    *common.ConversionInfo // Outgoing (source) conversion info for transfer
	ConvInfoIn     *common.ConversionInfo // Incoming (dest) conversion info for transfer
	Error          error
}
