package withdraw_test

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// NewValidWithdrawRequestedEvent returns a fully valid WithdrawRequestedEvent for use in tests.
func NewValidWithdrawRequestedEvent(userID, accountID, correlationID uuid.UUID, amount money.Money) *events.WithdrawRequestedEvent {
	return events.NewWithdrawRequestedEvent(
		userID,
		accountID,
		correlationID,
		events.WithWithdrawAmount(amount),
		events.WithWithdrawTimestamp(time.Now()),
	)
}

// NewValidWithdrawValidatedEvent returns a fully valid WithdrawValidatedEvent for use in tests.
func NewValidWithdrawValidatedEvent(userID, accountID, correlationID uuid.UUID, amount money.Money) *events.WithdrawValidatedEvent {
	// Create the validated event with the requested event embedded
	return events.NewWithdrawValidatedEvent(
		userID,
		accountID,
		correlationID,
		events.WithWithdrawRequestedEvent(*NewValidWithdrawRequestedEvent(userID, accountID, correlationID, amount)),
		events.WithTargetCurrency(amount.Currency().String()),
	)
}
