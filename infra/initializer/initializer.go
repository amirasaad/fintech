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
	if err := initializeExchangeRates(
		exchangeProvider,
		deps.ExchangeRateRegistry,
		cfg.ExchangeRateCache,
		logger,
	); err != nil {
		logger.Error("Failed to initialize exchange rates", "error", err)
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
		bus, err = infra_eventbus.NewWithRedis(cfg.Redis.URL, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis event bus: %w", err)
		}
	} else {
		bus = infra_eventbus.NewWithMemory(logger)
	}
	deps.EventBus = bus

	// Initialize payment provider with the checkout registry
	deps.PaymentProvider = stripepayment.NewStripePaymentProvider(
		bus,
		deps.CheckoutRegistry, // Use the checkout-specific registry
		cfg.PaymentProviders.Stripe,
		logger,
	)

	return
}

// initializeExchangeRates fetches and caches exchange rates during application startup
func initializeExchangeRates(
	exchangeRateProvider exchange.Exchange,
	registryProvider registry.Provider,
	cfg *config.ExchangeRateCache,
	logger *slog.Logger,
) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the exchange cache with the provided registry provider and config
	exchangeCache := caching.NewExchangeCache(
		registryProvider,
		logger,
		cfg,
	)

	// Check if cache is stale
	isStale, err := exchangeCache.IsCacheStale(ctx)
	if err != nil {
		logger.Warn("Failed to check cache status, will fetch new rates", "error", err)
	}

	// Only fetch new rates if cache is stale or non-existent
	if isStale {
		logger.Debug("Cache is stale, fetching new rates")
		// Fetch rates from the provider
		rates, err := exchangeRateProvider.FetchRates(
			ctx,
			"USD",
			exchangeRateProvider.SupportedPairs(),
		)
		if err != nil {
			return fmt.Errorf("failed to fetch exchange rates: %w", err)
		}

		// Cache the rates using ExchangeCache with exchange prefix
		if err := exchangeCache.CacheRates(
			ctx,
			rates,
			exchangeRateProvider.Metadata().Name,
		); err != nil {
			logger.Error("Failed to cache exchange rates", "error", err)
			return fmt.Errorf("failed to cache exchange rates: %w", err)
		}

		logger.Info("Successfully fetched and cached exchange rates",
			"provider", exchangeRateProvider.Metadata().Name,
			"rates_count", len(rates),
		)
	} else {
		logger.Info("Using cached exchange rates",
			"next_update_in", time.Until(time.Now().Add(cfg.TTL)),
		)
	}

	return nil
}
