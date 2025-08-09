// Package commands contains command DTOs for service and handler orchestration.
package commands

import "github.com/google/uuid"

// Deposit is a DTO for deposit operations (command pattern).
type Deposit struct {
	UserID      uuid.UUID
	AccountID   uuid.UUID
	Amount      float64
	Currency    string
	MoneySource string
	PaymentID   string
	Timestamp   int64
}
