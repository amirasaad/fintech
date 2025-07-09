package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/amirasaad/fintech/pkg/cache"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/provider"
)

// ExchangeRateAPIProvider implements the ExchangeRateProvider interface for exchangerate-api.com
// Updated to use v6 endpoint and config
type ExchangeRateAPIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	timeout    time.Duration
}

// ExchangeRateAPIResponseV6 represents the v6 response from the ExchangeRate API
// See: https://www.exchangerate-api.com/docs/standard-requests
// Example: { "result": "success", "documentation": "...", "terms_of_use": "...", "time_last_update_unix": 1585267200, ... }
type ExchangeRateAPIResponseV6 struct {
	Result             string             `json:"result"`
	Documentation      string             `json:"documentation"`
	TermsOfUse         string             `json:"terms_of_use"`
	TimeLastUpdateUnix int64              `json:"time_last_update_unix"`
	TimeLastUpdateUTC  string             `json:"time_last_update_utc"`
	TimeNextUpdateUnix int64              `json:"time_next_update_unix"`
	TimeNextUpdateUTC  string             `json:"time_next_update_utc"`
	BaseCode           string             `json:"base_code"`
	ConversionRates    map[string]float64 `json:"conversion_rates"`
	// Error fields (if any)
	ErrorType string `json:"error-type,omitempty"`
}

// NewExchangeRateAPIProvider creates a new ExchangeRate API provider using config
func NewExchangeRateAPIProvider(cfg config.ExchangeRateConfig, logger *slog.Logger) *ExchangeRateAPIProvider {
	return &ExchangeRateAPIProvider{
		apiKey:  cfg.ApiKey,
		baseURL: cfg.ApiUrl, // Should be like https://v6.exchangerate-api.com/v6
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		logger:  logger,
		timeout: cfg.HTTPTimeout,
	}
}

// FetchAndCacheRates fetches all rates for the base currency and caches them
func (p *ExchangeRateAPIProvider) FetchAndCacheRates(base string, cache cache.ExchangeRateCache, ttl time.Duration) error {
	url := fmt.Sprintf("%s/%s/latest/%s", p.baseURL, p.apiKey, base)
	p.logger.Info("Fetching exchange rates from API", "url", url)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ExchangeRateAPIResponseV6
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Result != "success" {
		return fmt.Errorf("API returned result=%s", apiResp.Result)
	}

	// Cache each rate as base:to
	for to, rate := range apiResp.ConversionRates {
		key := fmt.Sprintf("%s:%s", base, to)
		rateObj := &domain.ExchangeRate{
			FromCurrency: base,
			ToCurrency:   to,
			Rate:         rate,
			LastUpdated:  time.Now(),
			Source:       "exchangerate-api",
			ExpiresAt:    time.Now().Add(ttl),
		}
		if err := cache.Set(key, rateObj, ttl); err != nil {
			p.logger.Warn("Failed to cache exchange rate", "key", key, "error", err)
		}
	}
	p.logger.Info("Exchange rates cached successfully", "base", base, "count", len(apiResp.ConversionRates))
	return nil
}

// GetRate fetches the current exchange rate for a currency pair
func (p *ExchangeRateAPIProvider) GetRate(from, to string) (*domain.ExchangeRate, error) {
	// Update GetRate to use the v6 endpoint and response if needed, or rely on cache for POC
	// For now, we'll assume a simple call to the base URL with the API key
	url := fmt.Sprintf("%s/%s", p.baseURL, from)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ExchangeRateAPIResponseV6
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("API returned result=%s", apiResp.Result)
	}

	rate, exists := apiResp.ConversionRates[to]
	if !exists {
		return nil, fmt.Errorf("currency %s not found in response", to)
	}

	// Parse the date from the API response
	date, err := time.Parse("2006-01-02", "2006-01-02") // No date field in v6 response
	if err != nil {
		date = time.Now()
	}

	return &domain.ExchangeRate{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rate,
		LastUpdated:  time.Now(),
		Source:       "exchangerate-api",
		ExpiresAt:    date.Add(24 * time.Hour), // Rates typically valid for 24 hours
	}, nil
}

// GetRates fetches multiple exchange rates in a single request
func (p *ExchangeRateAPIProvider) GetRates(from string, to []string) (map[string]*domain.ExchangeRate, error) {
	// For this provider, we need to make a single request and extract the rates we need
	// We'll make a direct request to get all rates for the base currency

	// Since this provider returns all rates in one call, we need to make a full request
	url := fmt.Sprintf("%s/%s", p.baseURL, from)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ExchangeRateAPIResponseV6
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("API returned result=%s", apiResp.Result)
	}

	// Parse the date from the API response
	date, err := time.Parse("2006-01-02", "2006-01-02") // No date field in v6 response
	if err != nil {
		date = time.Now()
	}

	results := make(map[string]*domain.ExchangeRate)
	for _, currency := range to {
		if rate, exists := apiResp.ConversionRates[currency]; exists {
			results[currency] = &domain.ExchangeRate{
				FromCurrency: from,
				ToCurrency:   currency,
				Rate:         rate,
				LastUpdated:  time.Now(),
				Source:       "exchangerate-api",
				ExpiresAt:    date.Add(24 * time.Hour),
			}
		}
	}

	return results, nil
}

// Name returns the provider's name
func (p *ExchangeRateAPIProvider) Name() string {
	return "exchangerate-api"
}

// IsHealthy checks if the provider is currently available
func (p *ExchangeRateAPIProvider) IsHealthy() bool {
	// Make a simple health check request
	url := fmt.Sprintf("%s/USD", p.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode == http.StatusOK
}

// Ensure ExchangeRateAPIProvider implements provider.ExchangeRateProvider
var _ provider.ExchangeRateProvider = (*ExchangeRateAPIProvider)(nil)
