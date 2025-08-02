package events

var EventTypes = map[string]func() Event{
	"PaymentInitiated":            func() Event { return &PaymentInitiated{} },
	"PaymentCompleted":            func() Event { return &PaymentCompleted{} },
	"DepositRequested":            func() Event { return &DepositRequested{} },
	"DepositCurrencyConverted":    func() Event { return &DepositCurrencyConverted{} },
	"DepositBusinessValidated":    func() Event { return &DepositBusinessValidated{} },
	"DepositFailed":               func() Event { return &DepositFailed{} },
	"WithdrawRequested":           func() Event { return &WithdrawRequested{} },
	"WithdrawCurrencyConverted":   func() Event { return &WithdrawCurrencyConverted{} },
	"WithdrawBusinessValidated":   func() Event { return &WithdrawBusinessValidated{} },
	"WithdrawFailed":              func() Event { return &WithdrawFailed{} },
	"TransferRequested":           func() Event { return &TransferRequested{} },
	"TransferCurrencyConverted":   func() Event { return &TransferCurrencyConverted{} },
	"TransferBusinessValidated":   func() Event { return &TransferBusinessValidated{} },
	"TransferCompleted":           func() Event { return &TransferCompleted{} },
	"TransferFailed":              func() Event { return &TransferFailed{} },
	"CurrencyConversionRequested": func() Event { return &CurrencyConversionRequested{} },
	"CurrencyConverted":           func() Event { return &CurrencyConverted{} },
}
