package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
)

// DepositEventFactory creates a DepositBusinessValidationEvent.
type DepositEventFactory struct{}

func (f *DepositEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	convInfo *common.ConversionInfo,
	convertedMoney money.Money,
) (common.Event, error) {
	return events.DepositBusinessValidationEvent{
		DepositValidatedEvent: events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				FlowEvent: cre.FlowEvent,
				ID:        cre.ID,
				Amount:    cre.Amount,
				Timestamp: cre.Timestamp,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			FlowEvent:      cre.FlowEvent,
			ID:             cre.ID,
			RequestID:      cre.RequestID,
			TransactionID:  cre.TransactionID,
			Timestamp:      cre.Timestamp,
			ConversionInfo: convInfo,
		},
		Amount: convertedMoney,
	}, nil

}

// WithdrawEventFactory creates a WithdrawBusinessValidationEvent.
type WithdrawEventFactory struct{}

func (f *WithdrawEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	convInfo *common.ConversionInfo,
	convertedMoney money.Money,
) (common.Event, error) {
	// Create the event with proper FlowEvent initialization
	event := events.WithdrawBusinessValidationEvent{
		FlowEvent: cre.FlowEvent, // Initialize the top-level FlowEvent
		WithdrawValidatedEvent: events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: cre.FlowEvent,
				ID:        cre.ID,
				Amount:    cre.Amount,
				Timestamp: cre.Timestamp,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			FlowEvent:      cre.FlowEvent,
			ID:             cre.ID,
			RequestID:      cre.RequestID,
			TransactionID:  cre.TransactionID,
			Timestamp:      cre.Timestamp,
			ConversionInfo: convInfo,
		},
		Amount: convertedMoney,
	}
	return &event, nil
}

// TransferEventFactory creates a TransferBusinessValidationEvent.
type TransferEventFactory struct{}

func (f *TransferEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	convInfo *common.ConversionInfo,
	convertedMoney money.Money,
) (common.Event, error) {
	// Create the event using the factory function
	event := events.NewTransferBusinessValidationEvent(
		cre.UserID,
		cre.AccountID,
		cre.CorrelationID,
		events.WithTransferBusinessValidationAmount(convertedMoney),
		events.WithTransferValidatedEvent(events.TransferValidatedEvent{
			TransferRequestedEvent: events.TransferRequestedEvent{
				FlowEvent: cre.FlowEvent,
				ID:        cre.ID,
				Amount:    cre.Amount,
				Timestamp: cre.Timestamp,
			},
		}),
	)

	// Manually set the ConversionDoneEvent since it's not part of the options
	event.ConversionDoneEvent = events.ConversionDoneEvent{
		FlowEvent:      cre.FlowEvent,
		ID:             cre.ID,
		RequestID:      cre.RequestID,
		TransactionID:  cre.TransactionID,
		Timestamp:      cre.Timestamp,
		ConversionInfo: convInfo,
	}

	return event, nil
}
