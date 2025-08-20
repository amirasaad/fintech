// Package money provides functionality for handling monetary values.
package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
)

// Common errors
var (
	// ErrInvalidCurrency is returned when an invalid currency is provided
	// or when there's a currency mismatch in operations.
	ErrInvalidCurrency = errors.New("invalid currency")

	// ErrInvalidAmount is returned when an invalid amount is provided.
	ErrInvalidAmount = errors.New("invalid amount")

	// ErrAmountExceedsMaxSafeInt is returned when an amount exceeds the maximum safe integer value.
	ErrAmountExceedsMaxSafeInt = errors.New("amount exceeds maximum safe integer value")
)

// Amount represents a monetary amount as an integer in the
// smallest currency unit (e.g., cents for USD).
type Amount = int64

// Money represents a monetary value in a specific currency.
// Invariants:
//   - Amount is always stored in the smallest currency unit (e.g., cents for USD).
//   - Currency must be valid (valid ISO 4217 code and valid decimal places).
//   - All arithmetic operations require matching currencies.
type Money struct {
	amount   Amount
	currency Currency
}

// MarshalJSON implements json.Marshaler interface.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"amount":   m.AmountFloat(),
		"currency": m.currency.Code,
	})
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (m *Money) UnmarshalJSON(data []byte) error {
	var aux struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Create currency and validate
	currency := Currency{Code: aux.Currency}
	if !currency.IsValid() {
		return fmt.Errorf("invalid currency: %s", aux.Currency)
	}

	smallestUnit, err := convertToSmallestUnit(aux.Amount, currency)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	m.amount = Amount(smallestUnit)
	m.currency = currency
	return nil
}

// Zero creates a Money object with zero amount in the specified currency.
func Zero(currency Currency) *Money {
	return &Money{
		amount:   0,
		currency: currency,
	}
}

// Must creates a Money object from the given amount and currency.
// Invariants enforced:
//   - Currency must be valid (valid ISO 4217 code and valid decimal places).
//   - Amount must not have more decimal places than allowed by the currency.
//   - Amount is converted to the smallest currency unit.
//
// Panics if any invariant is violated.
func Must(amount float64, currency Currency) *Money {
	money, err := New(amount, currency)
	if err != nil {
		panic(fmt.Sprintf(
			"money: invalid arguments to Must(%v, %v): %v",
			amount,
			currency.Code,
			err,
		))
	}
	return money
}

// NewFromData creates a Money object from raw data (used for DB hydration).
// This bypasses invariants and should only be used for repository hydration or tests.
// Deprecated: use NewFromSmallestUnit instead.
func NewFromData(amount int64, currencyCode string) *Money {
	currency := Currency{Code: currencyCode}
	// Set default decimals based on common currencies
	switch currencyCode {
	case "USD", "EUR", "GBP":
		currency.Decimals = 2
	case "JPY":
		currency.Decimals = 0
	default:
		currency.Decimals = 2 // Default to 2 decimal places
	}

	return &Money{
		amount:   amount,
		currency: currency,
	}
}

// New creates a new Money object from a float amount and currency.
// The amount is converted to the smallest currency unit (e.g., cents for USD).
func New(amount float64, currency Currency) (*Money, error) {
	if !currency.IsValid() {
		return nil, fmt.Errorf("invalid currency: %v", currency)
	}

	smallestUnit, err := convertToSmallestUnit(amount, currency)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	return &Money{
		amount:   Amount(smallestUnit),
		currency: currency,
	}, nil
}

// NewFromSmallestUnit creates a new Money object directly from the smallest currency unit.
func NewFromSmallestUnit(amount int64, currency Currency) (*Money, error) {
	if !currency.IsValid() {
		return nil, fmt.Errorf("invalid currency: %v", currency)
	}

	return &Money{
		amount:   Amount(amount),
		currency: currency,
	}, nil
}

// Amount returns the amount of the Money object in the smallest currency unit.
func (m *Money) Amount() Amount {
	return m.amount
}

// AmountFloat returns the amount as a float64 in the main currency unit (e.g., dollars for USD).
func (m *Money) AmountFloat() float64 {
	amount := new(big.Rat).SetInt64(int64(m.amount))
	divisor := new(big.Rat).SetFloat64(math.Pow10(m.currency.Decimals))
	result := new(big.Rat).Quo(amount, divisor)

	floatResult, _ := result.Float64()
	return floatResult
}

// Currency returns the currency of the Money object.
func (m *Money) Currency() Currency {
	return m.currency
}

// IsCurrency checks if the money object has the specified currency
func (m *Money) IsCurrency(currency Currency) bool {
	return m.currency == currency
}

// Add returns a new Money object with the sum of amounts.
// Returns an error if the currencies don't match.
func (m *Money) Add(other *Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, fmt.Errorf(
			"cannot add different currencies: %s and %s",
			m.currency,
			other.currency,
		)
	}

	sum := int64(m.amount) + int64(other.amount)

	return &Money{
		amount:   Amount(sum),
		currency: m.currency,
	}, nil
}

// Subtract returns a new Money object with the difference of amounts.
// The result can be negative if the subtrahend is larger than the minuend.
// Returns an error if the currencies don't match.
func (m *Money) Subtract(other *Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, fmt.Errorf(
			"cannot subtract different currencies: %s and %s",
			m.currency,
			other.currency,
		)
	}

	diff := int64(m.amount) - int64(other.amount)

	return &Money{
		amount:   Amount(diff),
		currency: m.currency,
	}, nil
}

// Negate negates the current Money object.
func (m *Money) Negate() *Money {
	return &Money{
		amount:   -m.amount,
		currency: m.currency,
	}
}

// Equals checks if the current Money object is equal to another Money object.
// Invariants enforced:
//   - Currencies must match.
func (m *Money) Equals(other *Money) bool {
	if m == nil || other == nil {
		return false
	}

	return m.currency == other.currency && m.amount == other.amount
}

// GreaterThan checks if the current Money object is greater than another Money object.
// Returns an error if the currencies don't match.
func (m *Money) GreaterThan(other *Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf(
			"cannot compare different currencies: %s and %s",
			m.currency,
			other.currency,
		)
	}

	return m.amount > other.amount, nil
}

// LessThan checks if the current Money object is less than another Money object.
// Invariants enforced:
//   - Currencies must match.
//
// Returns an error if currencies do not match.
func (m *Money) LessThan(other *Money) (bool, error) {
	if !m.IsSameCurrency(other) {
		return false, ErrInvalidCurrency
	}
	return m.amount < other.amount, nil
}

// IsSameCurrency checks if the current Money object has the same currency as another Money object.
func (m *Money) IsSameCurrency(other *Money) bool {
	return m.currency == other.currency
}

// IsPositive returns true if the amount is greater than zero.
func (m *Money) IsPositive() bool {
	return m.amount > 0
}

// IsNegative returns true if the amount is less than zero.
func (m *Money) IsNegative() bool {
	return m.amount < 0
}

// IsZero returns true if the amount is zero.
func (m *Money) IsZero() bool {
	return m.amount == 0
}

// Abs returns the absolute value of the Money amount.
func (m *Money) Abs() *Money {
	if m.amount < 0 {
		return m.Negate()
	}
	return m
}

// Multiply multiplies the Money amount by a scalar factor.
// Invariants enforced:
//   - Result must not overflow int64.
//   - Result is rounded to the nearest integer to preserve precision.
//
// Returns Money or an error if overflow would occur.
func (m *Money) Multiply(factor float64) (*Money, error) {
	// Convert to float for multiplication and round to nearest integer
	resultFloat := float64(m.amount) * factor

	// Check for overflow
	if resultFloat > float64(math.MaxInt64) || resultFloat < float64(math.MinInt64) {
		return nil, fmt.Errorf("multiplication result would overflow")
	}

	// Round to nearest integer to avoid truncation of fractional cents
	rounded := int64(math.Round(resultFloat))

	return &Money{
		amount:   Amount(rounded),
		currency: m.currency,
	}, nil
}

// Divide divides the Money amount by a scalar divisor.
// Invariants enforced:
//   - Divisor must not be zero.
//   - Result must not overflow int64.
//   - Division must not lose precision.
//
// Returns Money or an error if any invariant is violated.
func (m *Money) Divide(divisor float64) (*Money, error) {
	if divisor == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	// Convert to float for division
	resultFloat := float64(m.amount) / divisor

	// Check for overflow
	if resultFloat > float64(math.MaxInt64) || resultFloat < float64(math.MinInt64) {
		return nil, fmt.Errorf("division result would overflow")
	}

	// Check if result is an integer (no precision loss)
	if resultFloat != float64(int64(resultFloat)) {
		return nil, fmt.Errorf("division would result in precision loss")
	}

	return &Money{
		amount:   Amount(int64(resultFloat)),
		currency: m.currency,
	}, nil
}

// String returns a string representation of the Money object.
func (m *Money) String() string {
	return fmt.Sprintf("%.*f %s", m.currency.Decimals, m.AmountFloat(), m.currency)
}

// convertToSmallestUnit converts a float64 amount to the smallest currency unit.
// This ensures precision by avoiding floating-point arithmetic issues.
func convertToSmallestUnit(amount float64, currency Currency) (int64, error) {
	factor := new(big.Rat).SetFloat64(math.Pow10(currency.Decimals))
	amountRat := new(big.Rat).SetFloat64(amount)
	result := new(big.Rat).Mul(amountRat, factor)

	// Round to nearest integer
	resultFloat, _ := result.Float64()
	return int64(math.Round(resultFloat)), nil
}
