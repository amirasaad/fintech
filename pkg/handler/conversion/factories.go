package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// DepositEventFactory creates a DepositConversionDoneEvent.
type DepositEventFactory struct{}

func (f *DepositEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	convInfo *common.ConversionInfo,
	convertedMoney money.Money,
) (domain.Event, error) {
	return events.DepositConversionDoneEvent{
		DepositValidatedEvent: events.DepositValidatedEvent{
			DepositRequestedEvent: events.DepositRequestedEvent{
				FlowEvent: cre.FlowEvent,
				ID:        cre.ID,
				Amount:    cre.FromAmount,
				Source:    "deposit",
				Timestamp: cre.Timestamp,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			FlowEvent:        cre.FlowEvent,
			ID:               uuid.New(),
			FromAmount:       cre.FromAmount,
			ToAmount:         convertedMoney,
			RequestID:        cre.RequestID,
			TransactionID:    cre.TransactionID,
			Timestamp:        cre.Timestamp,
			ConversionRate:   convInfo.ConversionRate,
			OriginalCurrency: cre.FromAmount.Currency().String(),
			ConvertedAmount:  convInfo.ConvertedAmount,
		},
		TransactionID: cre.TransactionID,
	}, nil
}

// WithdrawEventFactory creates a WithdrawConversionDoneEvent.
type WithdrawEventFactory struct{}

func (f *WithdrawEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	_ *common.ConversionInfo,
	convertedMoney money.Money,
) (domain.Event, error) {
	return events.WithdrawConversionDoneEvent{
		WithdrawValidatedEvent: events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: events.WithdrawRequestedEvent{
				FlowEvent: cre.FlowEvent,
				ID:        cre.ID,
				Amount:    cre.FromAmount,
				Timestamp: cre.Timestamp,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			// Populate fields as needed for Withdraw
		},
	}, nil
}

// TransferEventFactory creates a TransferConversionDoneEvent.
type TransferEventFactory struct{}

func (f *TransferEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	_ *common.ConversionInfo,
	convertedMoney money.Money,
) (domain.Event, error) {
	return events.TransferConversionDoneEvent{
		TransferValidatedEvent: events.TransferValidatedEvent{
			TransferRequestedEvent: events.TransferRequestedEvent{
				FlowEvent:      cre.FlowEvent,
				ID:             cre.ID,
				Amount:         cre.FromAmount,
				Source:         "transfer",
				DestAccountID:  cre.AccountID,
				ReceiverUserID: cre.UserID,
			},
		},
		ConversionDoneEvent: events.ConversionDoneEvent{
			// Populate fields as needed for Transfer
		},
	}, nil
}
