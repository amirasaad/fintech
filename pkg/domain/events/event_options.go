package events

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
)

// Deposit event options
func WithDepositSource(s string) func(e *DepositRequestedEvent) {
	return func(e *DepositRequestedEvent) { e.Source = s }
}

// Withdraw event options
func WithWithdrawAmount(m money.Money) func(e *WithdrawRequestedEvent) {
	return func(e *WithdrawRequestedEvent) { e.Amount = m }
}

func WithWithdrawExternalTarget(ext account.ExternalTarget) func(e *WithdrawRequestedEvent) {
	return func(e *WithdrawRequestedEvent) {
		e.BankAccountNumber = ext.BankAccountNumber
		e.RoutingNumber = ext.RoutingNumber
		e.ExternalWalletAddress = ext.ExternalWalletAddress
	}
}

// Transfer event options
func WithTransferSource(s string) func(e *TransferRequestedEvent) {
	return func(e *TransferRequestedEvent) { e.Source = s }
}

// Conversion event options
func WithConversionDoneAmount(m money.Money) func(e *ConversionDoneEvent) {
	return func(e *ConversionDoneEvent) { e.ConvertedAmount = m }
}

// Payment event options
func WithPaymentID(id string) func(e *PaymentCompletedEvent) {
	return func(e *PaymentCompletedEvent) { e.PaymentID = id }
}
