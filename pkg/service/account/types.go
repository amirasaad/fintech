// Package account provides business logic for account operations including creation, deposits, withdrawals, and balance inquiries.
package account

import "github.com/amirasaad/fintech/pkg/handler"

// OperationType represents the type of account operation
type OperationType = handler.OperationType

// Operation type constants for backward compatibility
const (
	OperationDeposit  = handler.OperationDeposit
	OperationWithdraw = handler.OperationWithdraw
	OperationTransfer = handler.OperationTransfer
)
