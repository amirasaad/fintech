package initializer

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"time"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/caching"
	infra_eventbus "github.com/amirasaad/fintech/infra/eventbus"
	exchangerateapi "github.com/amirasaad/fintech/infra/provider/exchangerateapi"
	stripepayment "github.com/amirasaad/fintech/infra/provider/stripepayment"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	currencyfixtures "github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider/exchange"

	"github.com/amirasaad/fintech/pkg/registry"
)

// loadCurrencyFixtures loads currency metadata from embedded CSV into the registry
func loadCurrencyFixtures(ctx context.Context, registry registry.Provider, logger *slog.Logger) {
	// Load currency metadata from embedded CSV
	logger.Info("Loading embedded currency metadata")
	_, filename, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(
		filepath.Dir(filename),
		"../../internal/fixtures/currency/meta.csv",
	)
	entities, err := currencyfixtures.LoadCurrencyMetaCSV(fixturePath)
	if err != nil {
		logger.Warn("Failed to load currency meta from CSV", "error", err)
		return
	}

	logger.Info("Loading currency meta from fixture",
		"to_register", len(entities))

	var registeredCount int
	for _, entity := range entities {
		if err := registry.Register(ctx, entity); err != nil {
			logger.Error("Failed to register currency", "code", entity.ID(), "error", err)
			// Continue with other currencies even if one fails
		} else {
			registeredCount++
		}
	}

	logger.Info("Successfully loaded currency fixtures", "registered_count", registeredCount)
}

// InitializeDependencies initializes all the application dependencies
func InitializeDependencies(cfg *config.App) (
	deps *app.Deps,
	err error,
) {
	// Load configuration
	deps = &app.Deps{}
	logger := setupLogger(cfg.Log)
	deps.Logger = logger

	// Initialize registry providers for each service
	deps.RegistryProvider, err = GetDefaultRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize main registry provider: %w", err)
	}

	// Initialize currency registry with dedicated provider
	deps.CurrencyRegistry, err = GetCurrencyRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize currency registry provider: %w", err)
	}

	ctx := context.Background()
	// Only load currency fixtures if the registry is empty
	count, err := deps.CurrencyRegistry.Count(ctx)
	if err != nil {
		logger.Warn("Failed to check currency registry count", "error", err)
	}

	if count == 0 {
		loadCurrencyFixtures(ctx, deps.CurrencyRegistry, logger)
	} else {
		logger.Info("Skipping currency fixtures load; registry not empty", "existing_count", count)
	}

	// Initialize checkout registry
	deps.CheckoutRegistry, err = GetCheckoutRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize checkout registry provider: %w", err)
	}

	// Initialize exchange rate registry
	deps.ExchangeRateRegistry, err = GetExchangeRateRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exchange rate registry provider: %w", err)
	}

	// Create the exchange rate provider
	exchangeProvider := exchangerateapi.NewExchangeRateAPIProvider(
		cfg.ExchangeRateAPIProviders.ExchangeRateApi,
		logger,
	)
	deps.ExchangeRateProvider = exchangeProvider

	// Initialize exchange rates
	if eerr := initializeExchangeRates(
		ctx,
		exchangeProvider,
		deps.ExchangeRateRegistry,
		cfg.ExchangeRateCache,
		logger,
	); eerr != nil {
		logger.Error("Failed to initialize exchange rates", "error", eerr)
		// Don't fail the entire startup for exchange rate initialization
	}

	// Initialize database
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		return nil, err
	}

	// Initialize unit of work
	deps.Uow = infra_repository.NewUoW(db)

	// Initialize event bus
	var bus eventbus.Bus
	if cfg.Redis.URL != "" {
		// Configure Redis event bus with DLQ retry settings
		busConfig := &infra_eventbus.RedisEventBusConfig{
			DLQRetryInterval: 5 * time.Minute, // Retry DLQ every 5 minutes
			DLQBatchSize:     10,              // Process 10 messages per batch
		}

		bus, err = infra_eventbus.NewWithRedis(cfg.Redis.URL, logger, busConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis event bus: %w", err)
		}
	} else {
		// Use in-memory bus for development
		bus = infra_eventbus.NewWithMemory(logger)
	}
	deps.EventBus = bus

	// Initialize payment provider with the checkout registry and unit of work
	deps.PaymentProvider = stripepayment.New(
		bus,
		deps.CheckoutRegistry, // Use the checkout-specific registry
		cfg.PaymentProviders.Stripe,
		logger,
		deps.Uow, // Pass the repository's UnitOfWork
	)

	return
}

// initializeExchangeRates fetches and caches exchange rates during application startup
// and sets up a background refresh mechanism
func initializeExchangeRates(
	ctx context.Context,
	exchangeRateProvider exchange.Exchange,
	registryProvider registry.Provider,
	cfg *config.ExchangeRateCache,
	logger *slog.Logger,
) error {
	// Start the background refresh goroutine
	go func(ctx context.Context, cacheLogger *slog.Logger) {
		// Initialize the exchange cache with the provided registry provider and config
		exchangeCache := caching.NewExchangeCache(
			registryProvider,
			cacheLogger,
			cfg,
		)

		// Set up periodic refresh
		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Check if cache is stale before refreshing
				isStale, timeUntilRefresh, err := exchangeCache.IsCacheStale(ctx)
				if err != nil {
					logger.Warn("Failed to check cache staleness", "error", err)
					continue
				}
				logger.Debug(
					"Cache staleness check",
					"is_stale", isStale,
					"time_until_refresh", timeUntilRefresh,
				)

				if isStale {
					logger.Info("Cache is stale, refreshing exchange rates")
					if err := refreshExchangeRates(
						ctx, exchangeRateProvider, exchangeCache, logger); err != nil {
						logger.Error("Failed to refresh exchange rates", "error", err)
					} else {
						logger.Info("Successfully refreshed exchange rates")
					}
				} else {
					logger.Debug(
						"Cache is still fresh, next refresh in",
						"duration", timeUntilRefresh,
					)
				}

			case <-ctx.Done():
				return
			}
		}
	}(ctx, logger)

	return nil
}

// refreshExchangeRates handles the actual refreshing of exchange rates
func refreshExchangeRates(
	ctx context.Context,
	exchangeRateProvider exchange.Exchange,
	exchangeCache *caching.ExchangeCache,
	logger *slog.Logger,
) error {
	// Create a timeout context for the refresh operation
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch rates from the provider
	rates, err := exchangeRateProvider.FetchRates(
		ctx,
		"USD", // Base currency
	)
	if err != nil {
		return fmt.Errorf("failed to fetch exchange rates: %w", err)
	}

	// Cache the rates using ExchangeCache
	if err := exchangeCache.CacheRates(
		ctx,
		rates,
		exchangeRateProvider.Metadata().Name,
	); err != nil {
		return fmt.Errorf("failed to cache exchange rates: %w", err)
	}

	logger.Info("Successfully cached exchange rates",
		"provider", exchangeRateProvider.Metadata().Name,
		"rates_count", len(rates),
	)
	return nil

}
