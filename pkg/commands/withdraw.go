// Package commands contains command DTOs for service and handler orchestration.
package commands

import "github.com/google/uuid"

// WithdrawCommand is a DTO for withdraw operations (command pattern).
type WithdrawCommand struct {
	UserID         uuid.UUID
	AccountID      uuid.UUID
	Amount         float64
	Currency       string
	MoneySource    string
	ExternalTarget *ExternalTarget // pointer for optionality
}

// ExternalTarget represents the destination for an external withdrawal, such as a bank account or wallet.
type ExternalTarget struct {
	BankAccountNumber     string
	RoutingNumber         string
	ExternalWalletAddress string
}
