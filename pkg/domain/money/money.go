package money

import (
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strings"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
)

// Use common.ErrInvalidCurrencyCode and common.ConversionInfo

// Amount represents a monetary amount as an integer in the smallest currency unit (e.g., cents for USD).
type Amount = int64

// Money represents a monetary value in a specific currency.
// Invariants:
//   - Amount is always stored in the smallest currency unit (e.g., cents for USD).
//   - Currency code must be valid ISO 4217 (3 uppercase letters).
//   - All arithmetic operations require matching currencies.
type Money struct {
	amount   Amount
	currency currency.Code
}

// NewMoney creates a new Money value object with the given amount and currency code.
// Invariants enforced:
//   - Currency code must be valid ISO 4217 (3 uppercase letters).
//   - Amount must not have more decimal places than allowed by the currency.
//   - Amount is converted to the smallest currency unit.
//
// Returns Money or an error if any invariant is violated.
func NewMoney(
	amount float64,
	currencyCode currency.Code,
) (
	money Money,
	err error,
) {
	if currencyCode == "" {
		currencyCode = currency.Code(currency.DefaultCurrency)
	}
	if !currency.IsValidCurrencyFormat(string(currencyCode)) {
		err = common.ErrInvalidCurrencyCode
		return
	}

	smallestUnit, err := convertToSmallestUnit(amount, string(currencyCode))
	if err != nil {
		return
	}

	money = Money{amount: Amount(smallestUnit), currency: currencyCode}
	return
}

// NewMoneyFromSmallestUnit creates a new Money object from the smallest currency unit.
// Invariants enforced:
//   - Currency code must be valid ISO 4217 (3 uppercase letters).
//
// Returns Money or an error if any invariant is violated.
func NewMoneyFromSmallestUnit(
	amount int64,
	currencyCode currency.Code,
) (
	money Money,
	err error,
) {
	if currencyCode == "" {
		currencyCode = currency.Code(currency.DefaultCurrency)
	}
	if !currency.IsValidCurrencyFormat(string(currencyCode)) {
		err = common.ErrInvalidCurrencyCode
		return
	}

	money = Money{amount: Amount(amount), currency: currencyCode}
	return
}

// Amount returns the amount of the Money object in the smallest currency unit.
func (m Money) Amount() Amount {
	return m.amount
}

// AmountFloat returns the amount as a float64 in the main currency unit (e.g., dollars for USD).
// Invariants enforced:
//   - Currency metadata must be valid.
func (m Money) AmountFloat() float64 {
	meta, err := currency.Get(string(m.currency))
	if err != nil {
		slog.Error("invalid currency code in AmountFloat", "currency", m.currency, "error", err)
		return 0
	}

	divisor := math.Pow10(meta.Decimals)
	return float64(m.amount) / divisor
}

// Currency returns the currency of the Money object.
func (m Money) Currency() currency.Code {
	return m.currency
}

// Add adds another Money object to the current Money object.
// Invariants enforced:
//   - Currencies must match.
//
// Returns Money or an error if currencies do not match.
func (m Money) Add(other Money) (Money, error) {
	if !m.IsSameCurrency(other) {
		return Money{}, common.ErrInvalidCurrencyCode
	}
	return Money{
		amount:   m.amount + other.amount,
		currency: m.currency,
	}, nil
}

// Subtract subtracts another Money object from the current Money object.
// Invariants enforced:
//   - Currencies must match.
//
// Returns Money or an error if currencies do not match.
func (m Money) Subtract(other Money) (Money, error) {
	if !m.IsSameCurrency(other) {
		return Money{}, common.ErrInvalidCurrencyCode
	}
	return Money{
		amount:   m.amount - other.amount,
		currency: m.currency,
	}, nil
}

// Negate negates the current Money object.
func (m Money) Negate() Money {
	return Money{
		amount:   -m.amount,
		currency: m.currency,
	}
}

// Equals checks if the current Money object is equal to another Money object.
// Invariants enforced:
//   - Currencies must match.
//
// Returns false if currencies do not match.
func (m Money) Equals(other Money) bool {
	return m.IsSameCurrency(other) && m.amount == other.amount
}

// GreaterThan checks if the current Money object is greater than another Money object.
// Invariants enforced:
//   - Currencies must match.
//
// Returns an error if currencies do not match.
func (m Money) GreaterThan(other Money) (bool, error) {
	if !m.IsSameCurrency(other) {
		return false, common.ErrInvalidCurrencyCode
	}
	return m.amount > other.amount, nil
}

// LessThan checks if the current Money object is less than another Money object.
// Invariants enforced:
//   - Currencies must match.
//
// Returns an error if currencies do not match.
func (m Money) LessThan(other Money) (bool, error) {
	if !m.IsSameCurrency(other) {
		return false, common.ErrInvalidCurrencyCode
	}
	return m.amount < other.amount, nil
}

// IsSameCurrency checks if the current Money object has the same currency as another Money object.
func (m Money) IsSameCurrency(other Money) bool {
	return m.currency == other.currency
}

// IsPositive returns true if the amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// IsNegative returns true if the amount is less than zero.
func (m Money) IsNegative() bool {
	return m.amount < 0
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// Abs returns the absolute value of the Money amount.
func (m Money) Abs() Money {
	if m.amount < 0 {
		return m.Negate()
	}
	return m
}

// Multiply multiplies the Money amount by a scalar factor.
// Invariants enforced:
//   - Result must not overflow int64.
//
// Returns Money or an error if overflow would occur.
func (m Money) Multiply(factor float64) (Money, error) {
	// Convert to float for multiplication
	resultFloat := float64(m.amount) * factor

	// Check for overflow
	if resultFloat > float64(math.MaxInt64) || resultFloat < float64(math.MinInt64) {
		return Money{}, fmt.Errorf("multiplication result would overflow")
	}

	return Money{
		amount:   Amount(int64(resultFloat)),
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
func (m Money) Divide(divisor float64) (Money, error) {
	if divisor == 0 {
		return Money{}, fmt.Errorf("division by zero")
	}

	// Convert to float for division
	resultFloat := float64(m.amount) / divisor

	// Check for overflow
	if resultFloat > float64(math.MaxInt64) || resultFloat < float64(math.MinInt64) {
		return Money{}, fmt.Errorf("division result would overflow")
	}

	// Check if result is an integer (no precision loss)
	if resultFloat != float64(int64(resultFloat)) {
		return Money{}, fmt.Errorf("division would result in precision loss")
	}

	return Money{
		amount:   Amount(int64(resultFloat)),
		currency: m.currency,
	}, nil
}

// String returns a string representation of the Money object.
func (m Money) String() string {
	meta, err := currency.Get(string(m.currency))
	if err != nil {
		slog.Error("invalid currency code in String", "currency", m.currency, "error", err)
		return ""
	}
	return fmt.Sprintf("%.*f %s", meta.Decimals, m.AmountFloat(), m.currency)
}

// convertToSmallestUnit converts a float64 amount to the smallest currency unit.
// This ensures precision by avoiding floating-point arithmetic issues.
func convertToSmallestUnit(amount float64, currencyCode string) (int64, error) {
	meta, err := currency.Get(currencyCode)
	if err != nil {
		return 0, err
	}
	// First, check if the amount has too many decimal places
	amountStr := fmt.Sprintf("%.10f", amount) // Use high precision for checking
	parts := strings.Split(amountStr, ".")
	if len(parts) > 1 {
		decimals := strings.TrimRight(parts[1], "0") // Remove trailing zeros
		if len(decimals) > meta.Decimals {
			return 0, fmt.Errorf("amount has more than %d decimal places", meta.Decimals)
		}
	}

	// Use big.Rat for precise decimal arithmetic
	amountStr = fmt.Sprintf("%.*f", meta.Decimals, amount)
	amountRat, ok := new(big.Rat).SetString(amountStr)
	if !ok {
		return 0, fmt.Errorf("invalid amount format: %f", amount)
	}

	multiplier := math.Pow10(meta.Decimals)
	smallestUnitRat := new(big.Rat).Mul(amountRat, big.NewRat(int64(multiplier), 1))

	if !smallestUnitRat.IsInt() {
		return 0, fmt.Errorf("amount has more than %d decimal places", meta.Decimals)
	}

	smallestUnit := smallestUnitRat.Num()
	if !smallestUnit.IsInt64() {
		return 0, fmt.Errorf("amount exceeds maximum safe integer value")
	}

	return smallestUnit.Int64(), nil
}
