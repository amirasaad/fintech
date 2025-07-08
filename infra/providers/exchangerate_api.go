package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/amirasaad/fintech/pkg/domain"
)

// ExchangeRateAPIProvider implements the ExchangeRateProvider interface for exchangerate-api.com
type ExchangeRateAPIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// ExchangeRateAPIResponse represents the response from the ExchangeRate API
type ExchangeRateAPIResponse struct {
	Success   bool               `json:"success"`
	Timestamp int64              `json:"timestamp"`
	Base      string             `json:"base"`
	Date      string             `json:"date"`
	Rates     map[string]float64 `json:"rates"`
}

// NewExchangeRateAPIProvider creates a new ExchangeRate API provider
func NewExchangeRateAPIProvider(apiKey string, logger *slog.Logger) *ExchangeRateAPIProvider {
	return &ExchangeRateAPIProvider{
		apiKey:  apiKey,
		baseURL: "https://api.exchangerate-api.com/v4/latest",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// GetRate fetches the current exchange rate for a currency pair
func (p *ExchangeRateAPIProvider) GetRate(from, to string) (*domain.ExchangeRate, error) {
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

	var apiResp ExchangeRateAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	rate, exists := apiResp.Rates[to]
	if !exists {
		return nil, fmt.Errorf("currency %s not found in response", to)
	}

	// Parse the date from the API response
	date, err := time.Parse("2006-01-02", apiResp.Date)
	if err != nil {
		date = time.Now()
	}

	return &domain.ExchangeRate{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rate,
		LastUpdated:  time.Unix(apiResp.Timestamp, 0),
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

	var apiResp ExchangeRateAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// Parse the date from the API response
	date, err := time.Parse("2006-01-02", apiResp.Date)
	if err != nil {
		date = time.Now()
	}

	results := make(map[string]*domain.ExchangeRate)
	for _, currency := range to {
		if rate, exists := apiResp.Rates[currency]; exists {
			results[currency] = &domain.ExchangeRate{
				FromCurrency: from,
				ToCurrency:   currency,
				Rate:         rate,
				LastUpdated:  time.Unix(apiResp.Timestamp, 0),
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
