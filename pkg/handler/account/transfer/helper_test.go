package transfer_test

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// NewValidTransferRequestedEvent returns a fully valid TransferRequestedEvent for use in tests.
func NewValidTransferRequestedEvent(userID, accountID, correlationID uuid.UUID, amount money.Money, toAccountID uuid.UUID) *events.TransferRequestedEvent {
	return events.NewTransferRequestedEvent(
		userID,
		accountID,
		correlationID,
		events.WithTransferRequestedAmount(amount),
		events.WithTransferDestAccountID(toAccountID),
	)
}

// NewValidTransferValidatedEvent returns a fully valid TransferValidatedEvent for use in tests.
func NewValidTransferValidatedEvent(userID, accountID, correlationID uuid.UUID, amount money.Money, toAccountID uuid.UUID) *events.TransferValidatedEvent {
	requestedEvent := NewValidTransferRequestedEvent(userID, accountID, correlationID, amount, toAccountID)
	return events.NewTransferValidatedEvent(
		userID,
		accountID,
		correlationID,
		events.WithTransferRequestedEvent(*requestedEvent),
	)
}

// NewMalformedTransferValidatedEvent returns a TransferValidatedEvent missing required fields (for edge case tests).
func NewMalformedTransferValidatedEvent(userID, accountID, correlationID, transactionID, destAccountID uuid.UUID, amount money.Money) *events.TransferValidatedEvent {
	// Create a malformed event with zero amount
	malformedAmount := money.Money{} // Zero money value
	requestedEvent := NewValidTransferRequestedEvent(userID, accountID, correlationID, malformedAmount, destAccountID)
	requestedEvent.ID = transactionID

	validatedEvent := events.NewTransferValidatedEvent(
		userID,
		accountID,
		correlationID,
		events.WithTransferRequestedEvent(*requestedEvent),
	)

	// Explicitly set the amount to zero to ensure it's malformed
	validatedEvent.Amount = money.Money{}
	return validatedEvent
}

// NewValidTransferDomainOpDoneEvent returns a fully valid TransferDomainOpDoneEvent for use in tests.
func NewValidTransferDomainOpDoneEvent(userID, accountID, correlationID, transactionID, destAccountID uuid.UUID, amount money.Money) *events.TransferDomainOpDoneEvent {
	validatedEvent := NewValidTransferValidatedEvent(userID, accountID, correlationID, amount, destAccountID)
	return &events.TransferDomainOpDoneEvent{
		TransferValidatedEvent: *validatedEvent,
		ConversionDoneEvent:    events.ConversionDoneEvent{ConvertedAmount: amount},
		TransactionID:          transactionID,
	}
}

// NewValidTransferBusinessValidatedEvent returns a fully valid TransferBusinessValidatedEvent for use in tests.
func NewValidTransferBusinessValidatedEvent(userID, accountID, correlationID, destAccountID uuid.UUID, amount money.Money) *events.TransferBusinessValidatedEvent {
	validatedEvent := NewValidTransferValidatedEvent(userID, accountID, correlationID, amount, destAccountID)
	return &events.TransferBusinessValidatedEvent{
		TransferValidatedEvent: *validatedEvent,
		ConversionDoneEvent:    events.ConversionDoneEvent{ConvertedAmount: amount},
	}
}

// NewValidTransferBusinessValidationEvent returns a fully valid TransferBusinessValidationEvent for use in tests.
func NewValidTransferBusinessValidationEvent(userID, accountID, correlationID, destAccountID uuid.UUID, amount money.Money) *events.TransferBusinessValidationEvent {
	// Create a validated event with all required fields
	validatedEvent := NewValidTransferValidatedEvent(userID, accountID, correlationID, amount, destAccountID)

	// Create the business validation event with proper nesting and all required fields
	event := events.NewTransferBusinessValidationEvent(
		userID,
		accountID,
		correlationID,
		events.WithTransferBusinessValidationAmount(amount),
		events.WithTransferValidatedEvent(*validatedEvent),
	)

	// Set the ConvertedAmount which is required by the handler
	event.ConvertedAmount = amount

	// Ensure all nested events have valid IDs
	if event.TransferRequestedEvent.ID == (uuid.UUID{}) {
		event.TransferRequestedEvent.ID = uuid.New()
	}

	return event
}

// NewValidTransferCompletedEvent returns a fully valid TransferCompletedEvent for use in tests.
func NewValidTransferCompletedEvent(userID, accountID, correlationID, txOutID, txInID, destAccountID uuid.UUID, amount money.Money) *events.TransferCompletedEvent {
	domainOpEvent := NewValidTransferDomainOpDoneEvent(userID, accountID, correlationID, txOutID, destAccountID, amount)
	return &events.TransferCompletedEvent{
		TransferDomainOpDoneEvent: *domainOpEvent,
		TxOutID:                   txOutID,
		TxInID:                    txInID,
	}
}

// NewValidTransferFailedEvent returns a fully valid TransferFailedEvent for use in tests.
func NewValidTransferFailedEvent(userID, accountID, correlationID, destAccountID uuid.UUID, amount money.Money, reason string) *events.TransferFailedEvent {
	requestedEvent := NewValidTransferRequestedEvent(userID, accountID, correlationID, amount, destAccountID)
	return &events.TransferFailedEvent{
		TransferRequestedEvent: *requestedEvent,
		Reason:                 reason,
	}
}

// NewInValidTransferRequestedEvent returns an invalid TransferRequestedEvent for testing validation failures.
func NewInValidTransferRequestedEvent(userID, accountID, correlationID uuid.UUID, money money.Money, uUID uuid.UUID) *events.TransferRequestedEvent {
	// Create a new event with invalid fields
	event := events.NewTransferRequestedEvent(
		userID,
		accountID,
		correlationID,
		events.WithTransferRequestedAmount(money),
		events.WithTransferDestAccountID(uuid.Nil),
	)

	// Override with invalid values
	event.FlowType = ""
	event.UserID = uuid.Nil
	event.AccountID = uuid.Nil
	event.CorrelationID = uuid.Nil
	event.ID = uUID
	event.DestAccountID = uuid.Nil

	return event
}

// NewMalformedTransferRequestedEvent returns a TransferRequestedEvent missing required fields (for edge case tests).
func NewMalformedTransferRequestedEvent() *events.TransferRequestedEvent {
	// Create a new event with no options to ensure minimal fields are set
	return &events.TransferRequestedEvent{
		FlowEvent: events.FlowEvent{},
		// Missing required fields
	}
}

// NewNegativeAmountTransferRequestedEvent returns a TransferRequestedEvent with negative amount.
func NewNegativeAmountTransferRequestedEvent(userID, accountID, correlationID, destAccountID uuid.UUID, currencyCode string) *events.TransferRequestedEvent {
	amount, _ := money.New(-100, currency.Code(currencyCode))
	return NewValidTransferRequestedEvent(userID, accountID, correlationID, amount, destAccountID)
}
