// Package money provides functionality for handling monetary values.
package money

// Currency represents a monetary unit with its standard decimal places
type Currency struct {
	Code     string // 3-letter ISO 4217 code (e.g., "USD")
	Decimals int    // Number of decimal places (0-18)
}

// Common currency instances
var (
	USD = Currency{"USD", 2} // US Dollar
	EUR = Currency{"EUR", 2} // Euro
	GBP = Currency{"GBP", 2} // British Pound
	JPY = Currency{"JPY", 0} // Japanese Yen
)

// DefaultCurrency is the default currency (USD)
var DefaultCurrency = USD

// IsValid checks if the currency is valid.
func (c Currency) IsValid() bool {
	if c.Decimals < 0 || c.Decimals > 18 {
		return false
	}
	return len(c.Code) == 3 &&
		c.Code[0] >= 'A' && c.Code[0] <= 'Z' &&
		c.Code[1] >= 'A' && c.Code[1] <= 'Z' &&
		c.Code[2] >= 'A' && c.Code[2] <= 'Z'
}

// String returns the currency code
func (c Currency) String() string { return c.Code }
