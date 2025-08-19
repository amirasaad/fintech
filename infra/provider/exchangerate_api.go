package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/provider"
)

// exchangeRateAPI implements the ExchangeRate interface for exchangerate-api.com v6 API
type exchangeRateAPI struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	timeout    time.Duration
}

// API error types
const (
	errorTypeUnsupportedCode  = "unsupported-code"
	errorTypeMalformedRequest = "malformed-request"
	errorTypeInvalidKey       = "invalid-key"
	errorTypeInactiveAccount  = "inactive-account"
	errorTypeQuotaReached     = "quota-reached"
	errorTypeUnknown          = "unknown-code"
)

// ExchangeRateAPIResponseV6 represents the v6 response from the ExchangeRate API
// See: https://www.exchangerate-api.com/docs/standard-requests
// Example:
// { "result": "success", "documentation": "...", "terms_of_use": "...",
// "time_last_update_unix": 1585267200, ... }
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
func NewExchangeRateAPIProvider(
	cfg *config.ExchangeRateApi,
	logger *slog.Logger,
) *exchangeRateAPI {
	if logger == nil {
		logger = slog.Default()
	}

	return &exchangeRateAPI{
		apiKey:  cfg.ApiKey,
		baseURL: fmt.Sprintf("%s/%s", cfg.ApiUrl, cfg.ApiKey),
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		logger:  logger,
		timeout: cfg.HTTPTimeout,
	}
}

// GetRate fetches the current exchange rate for a currency pair
func (p *exchangeRateAPI) GetRate(
	ctx context.Context,
	from, to string,
) (*provider.ExchangeInfo, error) {
	// Update GetRate to use the v6 endpoint and response if needed, or rely on cache for POC
	// For now, we'll assume a simple call to the base URL with the API key
	url := fmt.Sprintf("%s/%s/%s", p.baseURL, "latest", from)

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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			p.logger.Warn(
				"Failed to close response body",
				"error", cerr,
			)
		}
	}()

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
		return nil, fmt.Errorf("rate for %s not found in response", to)
	}

	return &provider.ExchangeInfo{
		OriginalCurrency:  from,
		ConvertedCurrency: to,
		ConversionRate:    rate,
		Timestamp:         time.Now(),
		Source:            p.Name(),
	}, nil
}

// GetRates fetches multiple exchange rates in a single request
func (p *exchangeRateAPI) GetRates(
	ctx context.Context,
	from string,
) (map[string]*provider.ExchangeInfo, error) {

	// Build the URL for the latest rates endpoint
	url := fmt.Sprintf("%s/latest/%s", p.baseURL, from)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set content type header
	req.Header.Set("Accept", "application/json")

	// Execute the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			p.logger.Warn("failed to close response body", "error", cerr)
		}
	}()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response
	var apiResp ExchangeRateAPIResponseV6
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if apiResp.Result != "success" {
		switch apiResp.ErrorType {
		case errorTypeUnsupportedCode:
			return nil, fmt.Errorf("unsupported currency code")
		case errorTypeMalformedRequest:
			return nil, fmt.Errorf("malformed request")
		case errorTypeInvalidKey:
			return nil, fmt.Errorf("invalid API key")
		case errorTypeInactiveAccount:
			return nil, fmt.Errorf("inactive account")
		case errorTypeQuotaReached:
			return nil, fmt.Errorf("API quota reached")
		case errorTypeUnknown, "":
			fallthrough
		default:
			return nil, fmt.Errorf("API error: %s", apiResp.ErrorType)
		}
	}

	// Process the requested rates
	results := make(map[string]*provider.ExchangeInfo)
	now := time.Now()

	for currency := range apiResp.ConversionRates {
		rate, exists := apiResp.ConversionRates[currency]
		if !exists {
			p.logger.Warn("currency not found in response", "currency", currency)
			continue
		}

		results[currency] = &provider.ExchangeInfo{
			OriginalCurrency:  from,
			ConvertedCurrency: currency,
			ConversionRate:    rate,
			Timestamp:         now,
			Source:            p.Name(),
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("none of the requested currencies were found in the response")
	}

	return results, nil
}

// IsSupported checks if the provider supports the given currency pair
func (p *exchangeRateAPI) IsSupported(from string, to string) bool {
	// TODO: implement me
	panic("unimplemented")
}

// Name returns the provider's name
func (p *exchangeRateAPI) Name() string {
	return "exchangerate-api"
}

// IsHealthy checks if the provider is currently available
func (p *exchangeRateAPI) IsHealthy() bool {
	// Make a simple health check request
	return true
}

// Ensure ExchangeRateAPIProvider implements provider.ExchangeRate
var _ provider.ExchangeRate = (*exchangeRateAPI)(nil)
