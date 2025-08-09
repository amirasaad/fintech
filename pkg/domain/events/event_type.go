package events

// EventType represents the type of an event in the system.
type EventType string

// Event type constants
const (
	// Payment events
	EventTypePaymentInitiated EventType = "Payment.Initiated"
	EventTypePaymentProcessed EventType = "Payment.Processed"
	EventTypePaymentCompleted EventType = "Payment.Completed"
	EventTypePaymentFailed    EventType = "Payment.Failed"

	// Deposit events
	EventTypeDepositRequested         EventType = "Deposit.Requested"
	EventTypeDepositCurrencyConverted EventType = "Deposit.CurrencyConverted"
	EventTypeDepositValidated         EventType = "Deposit.Validated"
	EventTypeDepositFailed            EventType = "Deposit.Failed"

	// Withdraw events
	EventTypeWithdrawRequested         EventType = "Withdraw.Requested"
	EventTypeWithdrawCurrencyConverted EventType = "Withdraw.CurrencyConverted"
	EventTypeWithdrawValidated         EventType = "Withdraw.Validated"
	EventTypeWithdrawFailed            EventType = "Withdraw.Failed"

	// Transfer events
	EventTypeTransferRequested         EventType = "Transfer.Requested"
	EventTypeTransferCurrencyConverted EventType = "Transfer.CurrencyConverted"
	EventTypeTransferValidated         EventType = "Transfer.Validated"
	EventTypeTransferCompleted         EventType = "Transfer.Completed"
	EventTypeTransferFailed            EventType = "Transfer.Failed"

	// Fee events
	EventTypeFeesCalculated EventType = "Fees.Calculated"

	// Currency conversion events
	EventTypeCurrencyConversionRequested EventType = "CurrencyConversion.Requested"
	EventTypeCurrencyConverted           EventType = "CurrencyConversion.Converted"
	EventTypeCurrencyConversionFailed    EventType = "CurrencyConversion.Failed"
)

// String returns the string representation of the event type.
func (et EventType) String() string {
	return string(et)
}
