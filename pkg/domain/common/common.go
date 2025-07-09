package common

import "errors"

// ErrInvalidCurrencyCode is returned when a currency code is invalid.
var ErrInvalidCurrencyCode = errors.New("invalid currency code") // Use error type if needed

// ConversionInfo holds details about a currency conversion performed during a transaction.
type ConversionInfo struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
}
