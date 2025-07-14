package common

import "errors"

// ErrInvalidCurrencyCode is returned when a currency code is invalid.
var ErrInvalidCurrencyCode = errors.New("invalid currency code") // Use error type if needed

// ErrInvalidDecimalPlaces is returned when a monetary amount has more decimal places than allowed by the currency.
var ErrInvalidDecimalPlaces = errors.New("amount has more decimal places than allowed by the currency")

var ErrAmountExceedsMaxSafeInt = errors.New("amount exceeds maximum safe integer value") // Deposit would overflow balance

// ConversionInfo holds details about a currency conversion performed during a transaction.
type ConversionInfo struct {
	OriginalAmount    float64
	OriginalCurrency  string
	ConvertedAmount   float64
	ConvertedCurrency string
	ConversionRate    float64
}

type Event any
