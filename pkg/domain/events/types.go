package events

// EventTypes maps event type constants to their respective constructor functions.
var EventTypes = map[EventType]func() Event{
	EventTypePaymentInitiated: func() Event { return &PaymentInitiated{} },
	EventTypePaymentCompleted: func() Event { return &PaymentCompleted{} },
	EventTypeDepositRequested: func() Event { return &DepositRequested{} },
	EventTypeDepositCurrencyConverted: func() Event {
		return &DepositCurrencyConverted{}
	},
	EventTypeDepositValidated:  func() Event { return &DepositValidated{} },
	EventTypeDepositFailed:     func() Event { return &DepositFailed{} },
	EventTypeWithdrawRequested: func() Event { return &WithdrawRequested{} },
	EventTypeWithdrawCurrencyConverted: func() Event {
		return &WithdrawCurrencyConverted{}
	},
	EventTypeWithdrawValidated: func() Event { return &WithdrawValidated{} },
	EventTypeWithdrawFailed:    func() Event { return &WithdrawFailed{} },
	EventTypeTransferRequested: func() Event { return &TransferRequested{} },
	EventTypeTransferCurrencyConverted: func() Event {
		return &TransferCurrencyConverted{}
	},
	EventTypeTransferValidated: func() Event { return &TransferValidated{} },
	EventTypeTransferCompleted: func() Event { return &TransferCompleted{} },
	EventTypeTransferFailed:    func() Event { return &TransferFailed{} },
	EventTypeCurrencyConversionRequested: func() Event {
		return &CurrencyConversionRequested{}
	},
	EventTypeCurrencyConverted: func() Event { return &CurrencyConverted{} },
	EventTypeCurrencyConversionFailed: func() Event {
		return &CurrencyConversionFailed{}
	},
}
