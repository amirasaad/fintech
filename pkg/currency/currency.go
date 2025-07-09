package currency

import (
	"fmt"
	"strconv"

	"github.com/amirasaad/fintech/pkg/registry"
)

const (
	// DefaultCurrency is the fallback currency code (USD)
	DefaultCurrency = "USD"
	// DefaultDecimals is the default number of decimal places for currencies
	DefaultDecimals = 2
)

// CurrencyMeta holds currency-specific metadata
type CurrencyMeta struct {
	Decimals int
	Symbol   string
}

// CurrencyRegistry wraps the generic registry for currency-specific operations
type CurrencyRegistry struct {
	registry *registry.Registry
}

// NewCurrencyRegistry creates a new currency registry with default currencies
func NewCurrencyRegistry() *CurrencyRegistry {
	cr := &CurrencyRegistry{
		registry: registry.NewRegistry(),
	}

	// Add default currencies
	defaultCurrencies := map[string]CurrencyMeta{
		"USD": {Decimals: 2, Symbol: "$"},
		"EUR": {Decimals: 2, Symbol: "€"},
		"JPY": {Decimals: 0, Symbol: "¥"},
		"KWD": {Decimals: 3, Symbol: "د.ك"},
		"EGP": {Decimals: 2, Symbol: "£"},
		"GBP": {Decimals: 2, Symbol: "£"},
		"CAD": {Decimals: 2, Symbol: "C$"},
		"AUD": {Decimals: 2, Symbol: "A$"},
		"CHF": {Decimals: 2, Symbol: "CHF"},
		"CNY": {Decimals: 2, Symbol: "¥"},
		"INR": {Decimals: 2, Symbol: "₹"},
	}

	for code, meta := range defaultCurrencies {
		cr.Register(code, meta)
	}

	return cr
}

// Register adds or updates a currency in the registry
func (cr *CurrencyRegistry) Register(code string, meta CurrencyMeta) {
	registryMeta := registry.Meta{
		ID:     code,
		Name:   code,
		Active: true,
		Metadata: map[string]string{
			"decimals": fmt.Sprintf("%d", meta.Decimals),
			"symbol":   meta.Symbol,
		},
	}
	cr.registry.Register(code, registryMeta)
}

// Get returns currency metadata for the given code
func (cr *CurrencyRegistry) Get(code string) CurrencyMeta {
	registryMeta := cr.registry.Get(code)

	// Extract currency-specific metadata
	decimals := DefaultDecimals
	if decStr, found := registryMeta.Metadata["decimals"]; found {
		if dec, err := strconv.Atoi(decStr); err == nil {
			decimals = dec
		}
	}

	symbol := code
	if sym, found := registryMeta.Metadata["symbol"]; found {
		symbol = sym
	}

	return CurrencyMeta{
		Decimals: decimals,
		Symbol:   symbol,
	}
}

// IsSupported checks if a currency code is registered
func (cr *CurrencyRegistry) IsSupported(code string) bool {
	return cr.registry.IsRegistered(code)
}

// ListSupported returns a list of all supported currency codes
func (cr *CurrencyRegistry) ListSupported() []string {
	return cr.registry.ListRegistered()
}

// Unregister removes a currency from the registry
func (cr *CurrencyRegistry) Unregister(code string) bool {
	return cr.registry.Unregister(code)
}

// Count returns the total number of registered currencies
func (cr *CurrencyRegistry) Count() int {
	return cr.registry.Count()
}

// Global currency registry instance
var globalCurrencyRegistry = NewCurrencyRegistry()

// Global convenience functions for currency operations
func Register(code string, meta CurrencyMeta) {
	globalCurrencyRegistry.Register(code, meta)
}

func Get(code string) CurrencyMeta {
	return globalCurrencyRegistry.Get(code)
}

func IsSupported(code string) bool {
	return globalCurrencyRegistry.IsSupported(code)
}

func ListSupported() []string {
	return globalCurrencyRegistry.ListSupported()
}

func Unregister(code string) bool {
	return globalCurrencyRegistry.Unregister(code)
}

func Count() int {
	return globalCurrencyRegistry.Count()
}
