package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
)

// DepositEventFactory creates a DepositCurrencyConverted event from a DepositRequested and CurrencyConverted event.
type DepositEventFactory struct {
	depositRequested *events.DepositRequested
}

func NewDepositEventFactory(depositRequested *events.DepositRequested) *DepositEventFactory {
	return &DepositEventFactory{
		depositRequested: depositRequested,
	}
}

func (f *DepositEventFactory) CreateNextEvent(convertedEvent *events.CurrencyConverted) common.Event {
	// Create the deposit currency converted event with the deposit request and converted amount
	dcc := events.NewDepositCurrencyConverted(convertedEvent)
	// Apply the deposit request details
	dcc.DepositRequested = *f.depositRequested
	return dcc
}

// WithdrawEventFactory creates a WithdrawCurrencyConverted event from a CurrencyConverted event.
type WithdrawEventFactory struct {
	withdrawRequested *events.WithdrawRequested
}

func NewWithdrawEventFactory(withdrawRequested *events.WithdrawRequested) *WithdrawEventFactory {
	return &WithdrawEventFactory{
		withdrawRequested: withdrawRequested,
	}
}

func (f *WithdrawEventFactory) CreateNextEvent(convertedEvent *events.CurrencyConverted) common.Event {
	// Create WithdrawCurrencyConverted with both the withdraw request and converted event
	return events.NewWithdrawCurrencyConverted(f.withdrawRequested, convertedEvent)
}

// TransferEventFactory creates a TransferCurrencyConverted event from a TransferRequested and CurrencyConverted event.
type TransferEventFactory struct {
	transferRequested *events.TransferRequested
}

func NewTransferEventFactory(transferRequested *events.TransferRequested) *TransferEventFactory {
	return &TransferEventFactory{
		transferRequested: transferRequested,
	}
}

func (f *TransferEventFactory) CreateNextEvent(convertedEvent *events.CurrencyConverted) common.Event {
	// Create TransferCurrencyConverted with the transfer request and apply the converted amount
	tcc := events.NewTransferCurrencyConverted(f.transferRequested)
	// Apply the converted amount and conversion info from the CurrencyConverted event
	tcc.CurrencyConverted = *convertedEvent
	return tcc
}
