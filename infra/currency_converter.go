package infra

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
)

// RealCurrencyConverter implements the CurrencyConverter interface using real exchange rates.
type RealCurrencyConverter struct {
	exchangeRateService *ExchangeRateService
	logger              *slog.Logger
	fallback            domain.CurrencyConverter
}

// NewRealCurrencyConverter creates a new real currency converter with fallback support.
func NewRealCurrencyConverter(exchangeRateService *ExchangeRateService, fallback domain.CurrencyConverter, logger *slog.Logger) *RealCurrencyConverter {
	return &RealCurrencyConverter{
		exchangeRateService: exchangeRateService,
		logger:              logger,
		fallback:            fallback,
	}
}

// Convert converts an amount from one currency to another using real exchange rates.
func (c *RealCurrencyConverter) Convert(amount float64, from, to string) (*domain.ConversionInfo, error) {
	if from == to {
		return &domain.ConversionInfo{
			OriginalAmount:    amount,
			OriginalCurrency:  from,
			ConvertedAmount:   amount,
			ConvertedCurrency: to,
			ConversionRate:    1.0,
		}, nil
	}

	// Try to get real exchange rate
	rate, err := c.exchangeRateService.GetRate(from, to)
	if err != nil {
		c.logger.Warn("Failed to get real exchange rate, falling back", "from", from, "to", to, "error", err)

		// Use fallback converter
		if c.fallback != nil {
			return c.fallback.Convert(amount, from, to)
		}

		return nil, domain.ErrExchangeRateUnavailable
	}

	convertedAmount := amount * rate.Rate

	c.logger.Info("Currency conversion completed",
		"from", from, "to", to, "amount", amount,
		"converted", convertedAmount, "rate", rate.Rate, "source", rate.Source)

	return &domain.ConversionInfo{
		OriginalAmount:    amount,
		OriginalCurrency:  from,
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: to,
		ConversionRate:    rate.Rate,
	}, nil
}

// GetRate returns the current exchange rate between two currencies.
func (c *RealCurrencyConverter) GetRate(from, to string) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	rate, err := c.exchangeRateService.GetRate(from, to)
	if err != nil {
		c.logger.Warn("Failed to get real exchange rate, falling back", "from", from, "to", to, "error", err)

		// Use fallback converter
		if c.fallback != nil {
			return c.fallback.GetRate(from, to)
		}

		return 0, domain.ErrExchangeRateUnavailable
	}

	return rate.Rate, nil
}

// IsSupported checks if a currency pair is supported by checking if we can get a rate.
func (c *RealCurrencyConverter) IsSupported(from, to string) bool {
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
