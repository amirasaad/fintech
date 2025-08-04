package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain/events"
)

// DepositEventFactory creates a DepositCurrencyConverted event from a CurrencyConverted event.
type DepositEventFactory struct{}

// CreateNextEvent creates DepositCurrencyConverted with converted event
func (f *DepositEventFactory) CreateNextEvent(
	cc *events.CurrencyConverted,
) events.Event {
	return events.NewDepositCurrencyConverted(cc)
}

// WithdrawEventFactory creates a WithdrawCurrencyConverted event from a CurrencyConverted event.
type WithdrawEventFactory struct{}

// CreateNextEvent creates WithdrawCurrencyConverted with converted event
func (f *WithdrawEventFactory) CreateNextEvent(
	cc *events.CurrencyConverted,
) events.Event {
	return events.NewWithdrawCurrencyConverted(cc)
}

// TransferEventFactory creates a TransferCurrencyConverted event from a CurrencyConverted event.
type TransferEventFactory struct{}

// CreateNextEvent creates TransferCurrencyConverted with converted event
func (f *TransferEventFactory) CreateNextEvent(
	cc *events.CurrencyConverted,
) events.Event {
	return events.NewTransferCurrencyConverted(cc)
}
