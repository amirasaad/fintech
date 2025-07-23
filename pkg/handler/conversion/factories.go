package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain"
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
) (domain.Event, error) {
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
) (domain.Event, error) {
	return events.WithdrawBusinessValidationEvent{
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
	}, nil
}

// TransferEventFactory creates a TransferConversionDoneEvent.
type TransferEventFactory struct{}

func (f *TransferEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	convInfo *common.ConversionInfo,
	convertedMoney money.Money,
) (domain.Event, error) {
	return events.TransferBusinessValidatedEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
			TransferRequestedEvent: events.TransferRequestedEvent{
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
