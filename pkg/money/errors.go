package money

import "errors"

// Common money package errors
var (
	// ErrMismatchedCurrencies is returned when performing operations on money with
	// different currencies
	ErrMismatchedCurrencies = errors.New("mismatched currencies")

	// ErrNegativeAmount is returned when an operation would result in a negative amount
	ErrNegativeAmount = errors.New("resulting amount cannot be negative")
)
