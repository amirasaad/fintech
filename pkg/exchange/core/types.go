package core

import "time"

// Rate represents an exchange rate between two currencies
type Rate struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Value     float64   `json:"value"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

// RateInfo contains additional metadata about a rate
type RateInfo struct {
	Rate
	IsStale bool          `json:"is_stale"`
	TTL     time.Duration `json:"ttl"`
}

// ConversionRequest represents a request to convert an amount between currencies
type ConversionRequest struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

// ConversionResult represents the result of a currency conversion
type ConversionResult struct {
	FromAmount float64 `json:"from_amount"`
	ToAmount   float64 `json:"to_amount"`
	Rate       float64 `json:"rate"`
	Source     string  `json:"source"`
}
