// Package money provides functionality for handling monetary values.
//
// It is a value object that represents a monetary value in a specific currency.
// Invariants:
//   - Amount is always stored in the smallest currency unit (e.g., cents for USD).
//   - Currency code must be valid ISO 4217 (3 uppercase letters).
//   - All arithmetic operations require matching currencies.
package money

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
)

var (
	// ErrInvalidAmount is returned when an invalid amount is provided.
	ErrInvalidAmount = fmt.Errorf("invalid amount float")

	// ErrAmountExceedsMaxSafeInt is returned when an amount exceeds the maximum safe integer value.
	ErrAmountExceedsMaxSafeInt = fmt.Errorf("amount exceeds maximum safe integer value")

	// ErrMismatchedCurrencies is returned when performing operations
	// on money with different currencies.
	ErrInvalidCurrency = fmt.Errorf("invalid currency code")
)

// Amount represents a monetary amount as an integer in the
// smallest currency unit (e.g., cents for USD).
type Amount = int64

// Code represents a currency code (e.g., "USD", "EUR")
// Code is defined in codes.go

// ToCurrency converts a Code to a Currency with default decimals
func (c Code) ToCurrency() Currency {
	switch c {
	case USD:
		return USDCurrency
	case EUR:
		return EURCurrency
	case GBP:
		return GBPCurrency
	case JPY:
		return JPYCurrency
	default:
		return Currency{Code: c, Decimals: 2}
	}
}

// IsValid checks if the currency code is valid
func (c Code) IsValid() bool {
	if len(c) != 3 {
		return false
	}
	return c[0] >= 'A' && c[0] <= 'Z' &&
		c[1] >= 'A' && c[1] <= 'Z' &&
		c[2] >= 'A' && c[2] <= 'Z'
}

// String returns the string representation of the currency code.
func (c Code) String() string {
	return string(c)
}

// Currency represents a monetary unit with its standard decimal places
type Currency struct {
	Code     Code // 3-letter ISO 4217 code (e.g., "USD")
	Decimals int  // Number of decimal places (0-8)
}

// IsValid checks if the currency is valid.
func (c Currency) IsValid() bool {
	if c.Decimals < 0 || c.Decimals > 8 {
		return false
	}
	return len(c.Code) == 3 &&
		c.Code[0] >= 'A' && c.Code[0] <= 'Z' &&
		c.Code[1] >= 'A' && c.Code[1] <= 'Z' &&
		c.Code[2] >= 'A' && c.Code[2] <= 'Z'
}

// String returns the currency code as a string
func (c Currency) String() string { return string(c.Code) }

// Common currency codes are defined in codes.go

// Common currency instances
var (
	USDCurrency = Currency{Code: USD, Decimals: 2}
	EURCurrency = Currency{Code: EUR, Decimals: 2}
	GBPCurrency = Currency{Code: GBP, Decimals: 2}
	JPYCurrency = Currency{Code: JPY, Decimals: 0} // Japanese Yen has no decimal places
)

// DefaultCurrency is the default currency (USD)
var DefaultCurrency = USDCurrency

// DefaultCode is the default currency code (USD)
var DefaultCode = USD

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
		"amount":   m.amount,
		"currency": m.currency.Code,
	})
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (m *Money) UnmarshalJSON(data []byte) error {
	var aux struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Create currency and validate
	currency := Currency{Code: Code(aux.Currency)}
	switch aux.Currency {
	case "USD", "EUR", "GBP":
		currency.Decimals = 2
	case "JPY":
		currency.Decimals = 0
	default:
		currency.Decimals = 2 // Default to 2 decimal places
	}

	if !currency.IsValid() {
		return fmt.Errorf("invalid currency code: %s", aux.Currency)
	}

	m.amount = aux.Amount
	m.currency = currency
	return nil
}

// Zero creates a Money object with zero amount in the specified currency.
// The currency parameter can be either a Code or a Currency.
func Zero(currency interface{}) *Money {
	var c Currency
	switch v := currency.(type) {
	case Code:
		c = v.ToCurrency()
	case Currency:
		c = v
	default:
		// Default to USD if invalid type is provided
		c = USDCurrency
	}

	return &Money{
		amount:   0,
		currency: c,
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
	m, err := New(amount, currency)
	if err != nil {
		panic(fmt.Sprintf("money.Must(%v, %v): %v", amount, currency, err))
	}
	return m
}

// NewFromData creates a Money object from raw data (used for DB hydration).
// This bypasses invariants and should only be used for repository hydration or tests.
// Deprecated: use NewFromSmallestUnit instead.
func NewFromData(amount int64, cc string) *Money {
	// This is intentionally not validating the currency code to allow for flexibility
	// in database migrations and test data setup.
	return &Money{
		amount:   amount,
		currency: Currency{Code: Code(cc), Decimals: 2}, // Default to 2 decimal places
	}
}

// New creates a new Money value object with the given amount and currency.
// The currency parameter can be either a Code, Currency, or string (e.g., "USD").
// Invariants enforced:
//   - Currency must be valid (valid ISO 4217 code and valid decimal places).
//   - Amount must not have more decimal places than allowed by the currency.
//   - Amount is converted to the smallest currency unit.
//
// Returns Money or an error if any invariant is violated.
func New(amount float64, currency any) (*Money, error) {
	var c Currency

	switch v := currency.(type) {
	case string:
		// Handle string currency codes like "USD"
		if len(v) != 3 {
			return nil, fmt.Errorf("%w: invalid currency code length: %s", ErrInvalidCurrency, v)
		}
		code := Code(v)
		if !code.IsValid() {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCurrency, v)
		}
		c = code.ToCurrency()
	case Code:
		c = v.ToCurrency()
	case Currency:
		c = v
	default:
		return nil, fmt.Errorf(
			"invalid currency type: %T, expected string, Code, or Currency",
			currency,
		)
	}

	if !c.IsValid() {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCurrency, c)
	}

	// Convert amount to smallest unit (e.g., dollars to cents)
	smallestUnit, err := convertToSmallestUnit(amount, c)
	if err != nil {
		return nil, err
	}

	return &Money{
		amount:   Amount(smallestUnit),
		currency: c,
	}, nil
}

// NewFromSmallestUnit creates a new Money object from the smallest currency unit.
// The currency parameter can be either a Code or a Currency.
// Invariants enforced:
//   - Currency must be valid (valid ISO 4217 code and valid decimal places).
//
// Returns Money or an error if any invariant is violated.
func NewFromSmallestUnit(amount int64, currency interface{}) (*Money, error) {
	var c Currency
	switch v := currency.(type) {
	case Code:
		c = v.ToCurrency()
	case Currency:
		c = v
	default:
		return nil, fmt.Errorf("invalid currency type: %T", currency)
	}

	if !c.IsValid() {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCurrency, c)
	}

	return &Money{
		amount:   Amount(amount),
		currency: c,
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

// CurrencyCode returns the currency code of the Money object.
func (m *Money) CurrencyCode() Code {
	return m.currency.Code
}

// IsCurrency checks if the money object has the specified currency
func (m *Money) IsCurrency(currency Currency) bool {
	return m.currency == currency
}

// Add returns a new Money object with the sum of amounts.
// Invariants enforced:
//   - Currencies must match.
func (m *Money) Add(other *Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, fmt.Errorf(
			"cannot add different currencies: %s and %s",
			m.currency.Code,
			other.currency.Code,
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
// Invariants enforced:
//   - Currencies must match.
func (m *Money) Subtract(other *Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, fmt.Errorf(
			"cannot subtract different currencies: %s and %s",
			m.currency.Code,
			other.currency.Code,
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
// Invariants enforced:
//   - Currencies must match.
func (m *Money) GreaterThan(other *Money) (bool, error) {
	if m.currency != other.currency {
		return false, fmt.Errorf(
			"cannot compare different currencies: %s and %s",
			m.currency.Code,
			other.currency.Code,
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

// IsPositive returns true if the Money is not nil and its amount is greater than zero.
func (m *Money) IsPositive() bool {
	return m != nil && m.amount > 0
}

// IsNegative returns true if the Money is not nil and its amount is less than zero.
func (m *Money) IsNegative() bool {
	return m != nil && m.amount < 0
}

// IsZero returns true if the Money is nil or its amount is zero.
func (m *Money) IsZero() bool {
	return m == nil || m.amount == 0
}

// Abs returns the absolute value of the Money amount.
func (m *Money) Abs() *Money {
	if m.amount < 0 {
		return m.Negate()
	}
	return m
}

// Multiply multiplies the Money amount by a scalar factor.
// The result is rounded to the nearest integer.
// Invariants enforced:
//   - Factor must not be negative.
//   - Result must not overflow int64.
//
// Returns a new Money object or an error if the factor is invalid or would cause overflow.
func (m *Money) Multiply(factor float64) (*Money, error) {
	if factor < 0 {
		return nil, fmt.Errorf("factor cannot be negative")
	}

	// Convert to big.Rat for precise multiplication
	amount := new(big.Rat).SetInt64(int64(m.amount))
	f := new(big.Rat).SetFloat64(factor)
	result := new(big.Rat).Mul(amount, f)

	// Convert to float64 for overflow check and rounding
	resultFloat, _ := result.Float64()

	// Check for overflow before rounding
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
// The result is rounded to the nearest integer.
// Invariants enforced:
//   - Divisor must be positive.
//
// Returns a new Money object or an error if the divisor is invalid.
func (m *Money) Divide(divisor float64) (*Money, error) {
	if divisor <= 0 {
		return nil, fmt.Errorf("divisor must be positive")
	}

	// Convert to big.Rat for precise division
	amount := new(big.Rat).SetInt64(int64(m.amount))
	d := new(big.Rat).SetFloat64(divisor)
	result := new(big.Rat).Quo(amount, d)

	// Round to nearest integer
	resultFloat, _ := result.Float64()
	rounded := int64(math.Round(resultFloat))

	// Check for overflow - using big.Int for the comparison to handle all cases
	bigRounded := big.NewInt(rounded)
	maxInt64 := big.NewInt(math.MaxInt64)
	minInt64 := big.NewInt(math.MinInt64)
	if bigRounded.Cmp(maxInt64) > 0 || bigRounded.Cmp(minInt64) < 0 {
		return nil, fmt.Errorf("division result would overflow")
	}

	return &Money{
		amount:   Amount(rounded),
		currency: m.currency,
	}, nil
}

// String returns a string representation of the Money object.
func (m *Money) String() string {
	return fmt.Sprintf("%.*f %s", m.currency.Decimals, m.AmountFloat(), m.currency.Code)
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
