package common

import "errors"

// ErrInvalidCurrencyCode is returned when a currency code is invalid.
var ErrInvalidCurrencyCode = errors.New("invalid currency code") // Use error type if needed

// ErrUnsupportedCurrency is return when currency is not supported by global registry #see: /pkg/currency
var ErrUnsupportedCurrency = errors.New("unsupported currency")

// ErrInvalidDecimalPlaces is returned when a monetary amount has more decimal places than allowed by the currency.
var ErrInvalidDecimalPlaces = errors.New("amount has more decimal places than allowed by the currency")

// ErrAmountExceedsMaxSafeInt is returned when an amount exceeds the maximum safe integer value.
var ErrAmountExceedsMaxSafeInt = errors.New("amount exceeds max safe int") // Deposit would overflow balance

// ConversionInfo holds details about a currency conversion performed during a transaction.
type ConversionInfo struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
}

// Event represents a domain event in the common package.
// The generic type T is used to specify the concrete event type.
type Event interface {
	// Type returns the reflect.Type of the concrete event.
	// This is used for type-safe event registration and dispatching.
	Type() string
}
