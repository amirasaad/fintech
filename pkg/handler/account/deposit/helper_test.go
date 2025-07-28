package deposit_test

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// NewValidDepositRequestedEvent creates a new DepositRequestedEvent for testing
func NewValidDepositRequestedEvent(userID, accountID, correlationID uuid.UUID, amount money.Money) *events.DepositRequestedEvent {
	return events.NewDepositRequestedEvent(
		userID,
		accountID,
		correlationID,
		events.WithDepositAmount(amount),
		events.WithDepositTimestamp(time.Now()),
	)
}

// NewValidDepositValidatedEvent creates a new DepositValidatedEvent for testing
func NewValidDepositValidatedEvent(userID, accountID, transactionID uuid.UUID, amount money.Money) *events.DepositValidatedEvent {
	// Create a new validated event with the transaction ID
	event := events.NewDepositValidatedEvent(
		userID,
		accountID,
		transactionID, // Use transactionID as correlation ID for testing
		events.WithDepositRequestedEvent(*NewValidDepositRequestedEvent(userID, accountID, transactionID, amount)),
	)

	// Ensure the TransactionID is set
	event.TransactionID = transactionID

	return event
}

// NewValidDepositBusinessValidationEvent creates a new DepositBusinessValidationEvent for testing
func NewValidDepositBusinessValidationEvent(userID, accountID, transactionID uuid.UUID, amount money.Money) *events.DepositBusinessValidationEvent {
	// Create a validated event with the transaction ID
	validatedEvent := NewValidDepositValidatedEvent(userID, accountID, transactionID, amount)

	// Create a new DepositBusinessValidationEvent with the validated event and amount
	event := events.NewDepositBusinessValidationEvent(
		userID,
		accountID,
		transactionID, // Use transactionID as correlation ID for testing
		events.WithDepositValidatedEvent(*validatedEvent),
		events.WithBusinessValidationAmount(amount),
	)

	// Set the FlowEvent fields directly
	event.FlowEvent = events.FlowEvent{
		FlowType:      "deposit",
		UserID:        userID,
		AccountID:     accountID,
		CorrelationID: transactionID,
	}

	return event
}

// NewValidDepositFailedEvent creates a new DepositFailedEvent for testing
func NewValidDepositFailedEvent(flowEvent events.FlowEvent, reason string) *events.DepositFailedEvent {
	return events.NewDepositFailedEvent(
		flowEvent,
		reason,
		events.WithDepositFailedTransactionID(uuid.New()),
	)
}

// NewValidDepositPersistedEvent creates a new DepositPersistedEvent for testing
func NewValidDepositPersistedEvent(userID, accountID, correlationID uuid.UUID, amount money.Money) *events.DepositPersistedEvent {
	validatedEvent := NewValidDepositValidatedEvent(userID, accountID, correlationID, amount)
	return events.NewDepositPersistedEvent(
		userID,
		accountID,
		correlationID,
		events.WithDepositValidatedEventForPersisted(*validatedEvent),
		events.WithTransactionIDForPersisted(uuid.New()),
		events.WithDepositAmountForPersisted(amount),
	)
}
