package events

import (
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
)

// CurrencyConversionRequested is emitted when a currency conversion is requested.
type CurrencyConversionRequested struct {
	DepositRequestedEvent
	EventID        uuid.UUID
	TransactionID  uuid.UUID
	AccountID      uuid.UUID
	UserID         uuid.UUID
	Amount         money.Money
	SourceCurrency string
	TargetCurrency string
	Timestamp      int64
}

// CurrencyConversionDone is emitted after a currency conversion is completed.
type CurrencyConversionDone struct {
	CurrencyConversionRequested
	ConvertedAmount money.Money            // converted amount and currency
	ConversionInfo  *common.ConversionInfo // details of conversion
}

// CurrencyConversionPersisted is emitted after conversion data is persisted.
type CurrencyConversionPersisted struct {
	CurrencyConversionDone
}

func (e CurrencyConversionRequested) EventType() string { return "CurrencyConversionRequested" }
func (e CurrencyConversionDone) EventType() string      { return "CurrencyConversionDone" }
func (e CurrencyConversionPersisted) EventType() string { return "CurrencyConversionPersisted" }
