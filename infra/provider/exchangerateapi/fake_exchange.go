package exchangerateapi

import (
	"context"
	"time"

	"github.com/amirasaad/fintech/pkg/provider/exchange"
)

type fakeExchangeRate struct {
}

func NewFakeExchangeRate() *fakeExchangeRate {
	return &fakeExchangeRate{}
}

// CheckHealth implements exchange.Exchange.
func (f *fakeExchangeRate) CheckHealth(ctx context.Context) error {
	return nil
}

// FetchRate implements exchange.Exchange.
func (f *fakeExchangeRate) FetchRate(
	ctx context.Context,
	from string,
	to string,
) (*exchange.RateInfo, error) {
	return &exchange.RateInfo{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         1,
		Timestamp:    time.Now(),
		Provider:     "fake",
	}, nil
}

// FetchRates implements exchange.Exchange.
func (f *fakeExchangeRate) FetchRates(
	ctx context.Context,
	from string,
) (map[string]*exchange.RateInfo, error) {
	return map[string]*exchange.RateInfo{
		"EUR": {
			FromCurrency: from,
			ToCurrency:   "EUR",
			Rate:         1,
			Timestamp:    time.Now(),
			Provider:     "fake",
		},
	}, nil
}

// IsSupported implements exchange.Exchange.
func (f *fakeExchangeRate) IsSupported(from string, to string) bool {
	return true
}

// SupportedPairs implements exchange.Exchange.
func (f *fakeExchangeRate) SupportedPairs() []string {
	return []string{"EUR"}
}

func (f *fakeExchangeRate) Metadata() exchange.ProviderMetadata {
	return exchange.ProviderMetadata{
		Name:    "fake",
		Version: "v1",
	}
}

var _ exchange.Exchange = &fakeExchangeRate{}
