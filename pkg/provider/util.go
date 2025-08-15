package provider

import (
	"context"
	"errors"
	"sync"
	"time"
)

// HealthCheckAll checks the health of all providers and returns a map of results
func HealthCheckAll(
	ctx context.Context,
	providers []HealthChecker,
) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range providers {
		wrapped, ok := p.(interface{ Metadata() ProviderMetadata })
		if !ok {
			continue
		}

		wg.Add(1)
		go func(p HealthChecker, name string) {
			defer wg.Done()

			err := p.CheckHealth(ctx)
			mu.Lock()
			results[name] = err
			mu.Unlock()
		}(p, wrapped.Metadata().Name)
	}

	wg.Wait()
	return results
}

// FirstHealthy returns the first healthy provider from the list
func FirstHealthy(
	ctx context.Context,
	providers []HealthChecker,
) (HealthChecker, error) {
	for _, p := range providers {
		if p.CheckHealth(ctx) == nil {
			return p, nil
		}
	}
	return nil, errors.New("no healthy providers available")
}

// AllHealthy checks if all providers are healthy
func AllHealthy(
	ctx context.Context,
	providers []HealthChecker,
) error {
	var errs []error

	for _, p := range providers {
		if err := p.CheckHealth(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// RateStats contains statistics about exchange rates
type RateStats struct {
	Min     float64
	Max     float64
	Average float64
	Count   int
}

// CalculateStats calculates statistics for a set of rates
func CalculateStats(rates []float64) RateStats {
	if len(rates) == 0 {
		return RateStats{}
	}

	min := rates[0]
	max := rates[0]
	sum := 0.0

	for _, rate := range rates {
		if rate < min {
			min = rate
		}
		if rate > max {
			max = rate
		}
		sum += rate
	}

	return RateStats{
		Min:     min,
		Max:     max,
		Average: sum / float64(len(rates)),
		Count:   len(rates),
	}
}

// RateHistory tracks historical rate data
type RateHistory struct {
	rates []RateInfo
	mu    sync.RWMutex
	size  int
}

// NewRateHistory creates a new RateHistory with the specified maximum size
func NewRateHistory(size int) *RateHistory {
	return &RateHistory{
		rates: make([]RateInfo, 0, size),
		size:  size,
	}
}

// Add adds a new rate to the history
func (h *RateHistory) Add(rate RateInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.rates = append(h.rates, rate)

	// Trim the slice if it exceeds the maximum size
	if len(h.rates) > h.size {
		h.rates = h.rates[len(h.rates)-h.size:]
	}
}

// Get returns the rate history
func (h *RateHistory) Get() []RateInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rates := make([]RateInfo, len(h.rates))
	copy(rates, h.rates)
	return rates
}

// Average calculates the average rate over the specified duration
func (h *RateHistory) Average(since time.Time) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var sum float64
	var count int

	for _, rate := range h.rates {
		if rate.Timestamp.After(since) {
			sum += rate.Rate
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}
