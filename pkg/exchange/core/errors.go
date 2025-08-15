package core

import "errors"

// Common errors for exchange operations
var (
	// ErrInvalidRate indicates that an invalid exchange rate was provided
	ErrInvalidRate = errors.New("invalid exchange rate")

	// ErrUnsupportedCurrencyPair indicates that the currency pair is not supported
	ErrUnsupportedCurrencyPair = errors.New("unsupported currency pair")

	// ErrProviderUnavailable indicates that the rate provider is not available
	ErrProviderUnavailable = errors.New("rate provider unavailable")

	// ErrRateNotFound indicates that the requested rate was not found
	ErrRateNotFound = errors.New("exchange rate not found")

	// ErrInvalidAmount indicates that an invalid amount was provided
	ErrInvalidAmount = errors.New("invalid amount")
)

// ProviderError represents an error from a rate provider
type ProviderError struct {
	Provider string
	Err      error
}

func (e *ProviderError) Error() string {
	return "provider " + e.Provider + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// IsProviderError checks if an error is a ProviderError
func IsProviderError(err error) bool {
	_, ok := err.(*ProviderError)
	return ok
}
