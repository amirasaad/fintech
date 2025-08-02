package events

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// CurrencyConversionRequested is an agnostic event for requesting currency conversion in any business flow.
type CurrencyConversionRequested struct {
	FlowEvent
	Amount        money.Money
	To            currency.Code
	TransactionID uuid.UUID
}

func (e CurrencyConversionRequested) Type() string { return "CurrencyConversionRequested" }

// CurrencyConverted is an agnostic event for reporting the successful result of a currency conversion.
type CurrencyConverted struct {
	FlowEvent
	TransactionID   uuid.UUID
	ConvertedAmount money.Money
	ConversionInfo  *common.ConversionInfo
}

func (e CurrencyConverted) Type() string { return "CurrencyConverted" }

// CurrencyConversionFailed is an event for reporting a failed currency conversion.
type CurrencyConversionFailed struct {
	FlowEvent
	TransactionID uuid.UUID
	Amount        money.Money
	To            currency.Code
	Reason        string
}

func (e CurrencyConversionFailed) Type() string { return "CurrencyConversionFailed" }
