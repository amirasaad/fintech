package provider

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"

	"github.com/amirasaad/fintech/pkg/domain"
)

// ExchangeRateCurrencyConverter implements the CurrencyConverter
// interface using ExchangeRate v6 API.
type ExchangeRateCurrencyConverter struct {
	exchangeRateService *ExchangeRateService
	logger              *slog.Logger
	fallback            domain.CurrencyConverter
}

// NewExchangeRateCurrencyConverter creates
// a new ExchangeRateCurrencyConverter with fallback support.
func NewExchangeRateCurrencyConverter(
	exchangeRateService *ExchangeRateService,
	fallback domain.CurrencyConverter,
	logger *slog.Logger,
) *ExchangeRateCurrencyConverter {
	return &ExchangeRateCurrencyConverter{
		exchangeRateService: exchangeRateService,
		logger:              logger,
		fallback:            fallback,
	}
}

// Convert converts an amount from one currency to another using ExchangeRate v6 API.
func (c *ExchangeRateCurrencyConverter) Convert(
	amount float64,
	from currency.Code,
	to currency.Code,
) (*currency.Info, error) {
	if from == to {
		return &currency.Info{
			OriginalAmount:    amount,
			OriginalCurrency:  from.String(),
			ConvertedAmount:   amount,
			ConvertedCurrency: to.String(),
			ConversionRate:    1.0,
		}, nil
	}

	// Try to get real exchange rate
	rate, err := c.exchangeRateService.GetRate(from.String(), to.String())
	if err != nil {
		c.logger.Warn(
			"Failed to get real exchange rate, falling back",
			"from", from,
			"to", to,
			"error", err,
		)

		// Use fallback converter
		if c.fallback != nil {
			return c.fallback.Convert(amount, from, to)
		}

		return nil, domain.ErrExchangeRateUnavailable
	}

	convertedAmount := amount * rate.Rate

	c.logger.Info(
		"Currency conversion completed",
		"from", from,
		"to", to,
		"amount", amount,
		"converted", convertedAmount,
		"rate", rate.Rate,
		"source", rate.Source,
	)

	return &currency.Info{
		OriginalAmount:    amount,
		OriginalCurrency:  from.String(),
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: to.String(),
		ConversionRate:    rate.Rate,
	}, nil
}

// GetRate returns the current exchange rate between two currencies
// using ExchangeRate v6 API.
func (c *ExchangeRateCurrencyConverter) GetRate(
	from, to string,
) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	rate, err := c.exchangeRateService.GetRate(from, to)
	if err != nil {
		c.logger.Warn(
			"Failed to get real exchange rate, falling back",
			"from", from,
			"to", to,
			"error", err,
		)

		// Use fallback converter
		if c.fallback != nil {
			return c.fallback.GetRate(from, to)
		}

		return 0, domain.ErrExchangeRateUnavailable
	}

	return rate.Rate, nil
}

// IsSupported checks if a currency pair is supported
//
//	by checking if we can get a rate using ExchangeRate v6 API.
func (c *ExchangeRateCurrencyConverter) IsSupported(
	from, to string,
) bool {
	if from == to {
		return true
	}

	// Try to get a rate to check if supported
	_, err := c.exchangeRateService.GetRate(from, to)
	if err == nil {
		return true
	}

	// Check fallback
	if c.fallback != nil {
		return c.fallback.IsSupported(from, to)
	}

	return false
}
