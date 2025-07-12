// Package account provides business logic for account operations including creation, deposits, withdrawals, and balance inquiries.
package account

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	mon "github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// OperationType represents the type of account operation
type OperationType string

const (
	OperationDeposit  OperationType = "deposit"
	OperationWithdraw OperationType = "withdraw"
)

// operationRequest contains the common parameters for account operations
type operationRequest struct {
	userID       uuid.UUID
	accountID    uuid.UUID
	amount       float64
	currencyCode currency.Code
	operation    OperationType
}

// operationResult contains the result of an account operation
type operationResult struct {
	transaction *account.Transaction
	convInfo    *common.ConversionInfo
}

// operationHandler defines the interface for handling account operations
type operationHandler interface {
	execute(account *account.Account, userID uuid.UUID, money mon.Money) (*account.Transaction, error)
}