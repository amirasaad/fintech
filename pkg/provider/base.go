package provider

import (
	"context"
	"fmt"
	"time"
)

// BaseProvider is a base implementation of the Provider interface
// that can be embedded in concrete provider implementations
type BaseProvider struct {
	metadata     ProviderMetadata
	supported    map[string]bool
	supportedAll []string
}

// GetRate implements ExchangeRate.
func (p *BaseProvider) GetRate(ctx context.Context, from string, to string) (*ExchangeInfo, error) {
	panic("unimplemented")
}

// GetRates implements ExchangeRate.
func (p *BaseProvider) GetRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*ExchangeInfo, error) {
	panic("unimplemented")
}

// IsHealthy implements ExchangeRate.
func (p *BaseProvider) IsHealthy() bool {
	panic("unimplemented")
}

// Name implements ExchangeRate.
func (p *BaseProvider) Name() string {
	panic("unimplemented")
}

// NewBaseProvider creates a new BaseProvider with the given name and version
func NewBaseProvider(name, version string, supportedPairs []string) *BaseProvider {
	supported := make(map[string]bool, len(supportedPairs))
	for _, pair := range supportedPairs {
		supported[pair] = true
	}

	return &BaseProvider{
		metadata: ProviderMetadata{
			Name:        name,
			Version:     version,
			LastUpdated: time.Now(),
			IsActive:    true,
		},
		supported:    supported,
		supportedAll: supportedPairs,
	}
}

// FetchRate implements the RateFetcher interface
func (p *BaseProvider) FetchRate(
	ctx context.Context,
	from, to string,
) (*RateInfo, error) {
	return nil, fmt.Errorf("FetchRate not implemented")
}

// FetchRates implements the RateFetcher interface
func (p *BaseProvider) FetchRates(
	ctx context.Context,
	from string,
	to []string,
) (map[string]*RateInfo, error) {
	return nil, fmt.Errorf("FetchRates not implemented")
}

// CheckHealth implements the HealthChecker interface
func (p *BaseProvider) CheckHealth(ctx context.Context) error {
	// Default implementation assumes the provider is always healthy
	return nil
}

// IsSupported implements the SupportedChecker interface
func (p *BaseProvider) IsSupported(from, to string) bool {
	key := fmt.Sprintf("%s_%s", from, to)
	_, ok := p.supported[key]
	return ok
}

// SupportedPairs implements the SupportedChecker interface
func (p *BaseProvider) SupportedPairs() []string {
	// Return a copy to prevent external modifications
	pairs := make([]string, len(p.supportedAll))
	copy(pairs, p.supportedAll)
	return pairs
}

// Metadata implements the Provider interface
func (p *BaseProvider) Metadata() ProviderMetadata {
	return p.metadata
}

// UpdateMetadata updates the provider's metadata
func (p *BaseProvider) UpdateMetadata(updates ProviderMetadata) {
	p.metadata = updates
}
