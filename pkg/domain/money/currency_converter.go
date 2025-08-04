package money

import (
	"errors"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/common"
)

var (
	// ErrExchangeRateUnavailable indicates the exchange rate service is unreachable or down.
	ErrExchangeRateUnavailable = errors.New("exchange rate service unavailable")

	// ErrUnsupportedCurrencyPair indicates the currency pair is not supported.
	ErrUnsupportedCurrencyPair = errors.New("unsupported currency pair")

	// ErrExchangeRateExpired indicates the exchange rate data is stale or expired.
	ErrExchangeRateExpired = errors.New("exchange rate has expired")

	// ErrExchangeRateInvalid indicates the received exchange rate is invalid.
	ErrExchangeRateInvalid = errors.New("invalid exchange rate received")
)

// CurrencyConverter defines the interface for converting amounts between currencies.
type CurrencyConverter interface {
	// Convert converts an amount from one currency to another.
	// Returns the converted amount and the rate used, or an error if conversion is not possible.
	Convert(amount float64, from, to string) (*common.ConversionInfo, error)

	// GetRate returns the current exchange rate between two currencies.
	// This is useful for displaying rates without performing a conversion.
	GetRate(from, to string) (float64, error)

	// IsSupported checks if a currency pair is supported by the converter.
	IsSupported(from, to string) bool
}

// ExchangeRate represents a single exchange rate with metadata.
type ExchangeRate struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
	LastUpdated  time.Time
	Source       string
	ExpiresAt    time.Time
}

// ConvertMoney converts a Money value object to the target currency using the converter.
// Returns a new Money object in the target currency and the conversion info.
func ConvertMoney(
	converter CurrencyConverter,
	m Money,
	to string,
) (Money, *common.ConversionInfo, error) {
	convInfo, err := converter.Convert(m.AmountFloat(), m.Currency().String(), to)
	if err != nil {
		return Money{}, nil, err
	}
	converted, err := New(convInfo.ConvertedAmount, m.Currency())
	if err != nil {
		return Money{}, nil, err
	}
	return converted, convInfo, nil
}
