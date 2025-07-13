// Package account provides business logic for account operations including creation, deposits, withdrawals, and balance inquiries.
package account

// OperationType represents the type of account operation
type OperationType string

// Operation type constants define the types of account operations.
const (
	OperationDeposit  OperationType = "deposit"
	OperationWithdraw OperationType = "withdraw"
	OperationTransfer OperationType = "transfer"
)
