package initializer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/infra/caching"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"

	"github.com/amirasaad/fintech/infra"
	infra_eventbus "github.com/amirasaad/fintech/infra/eventbus"
	infra_provider "github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	currencyfixtures "github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

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
	deps.RegistryProvider, err = GetRegistryProvider("app", cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize main registry provider: %w", err)
	}

	currencyRegistry, err := GetRegistryProvider("currency", cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize currency registry provider: %w", err)
	}
	deps.CurrencyRegistry = currencyRegistry

	ctx := context.Background()
	// Only load currency fixtures if the registry is empty
	count, err := currencyRegistry.Count(ctx)
	if err != nil {
		logger.Warn("Failed to check currency registry count", "error", err)
	}

	// Load currency metadata from CSV
	entities, err := currencyfixtures.LoadCurrencyMetaCSV(
		"../../internal/fixtures/currency/meta.csv")
	if err != nil {
		logger.Warn("Failed to load currency meta from CSV", "error", err)
	}

	logger.Info("Loading currency meta from fixture", "count", count)
	for _, entity := range entities {
		if err := currencyRegistry.Register(ctx, entity); err != nil {
			logger.Error("Failed to register currency", "code", entity.ID(), "error", err)
			// Continue with other currencies even if one fails
		}
	}
	logger.Info("Successfully loaded currency fixtures", "count", count)

	deps.CheckoutRegistry, err = GetRegistryProvider("checkout", cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize checkout registry provider: %w", err)
	}

	// Initialize exchange rate registry
	exchangeRateRegistry, err := GetRegistryProvider("exchange_rate", cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exchange rate registry provider: %w", err)
	}
	deps.ExchangeRateRegistry = exchangeRateRegistry

	// Create the exchange rate provider
	exchangeProvider := infra_provider.NewExchangeRateAPIProvider(
		cfg.ExchangeRateAPIProviders.ExchangeRateApi,
		logger,
	)
	if err := initializeExchangeRates(exchangeProvider, exchangeRateRegistry, logger); err != nil {
		logger.Error("Failed to initialize exchange rates", "error", err)
		// Don't fail the entire startup for exchange rate initialization
	}
	deps.ExchangeRateProvider = exchangeProvider

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

	// Initialize payment provider
	deps.PaymentProvider = infra_provider.NewStripePaymentProvider(
		bus,
		deps.RegistryProvider, // Use the single registry provider
		cfg.PaymentProviders.Stripe,
		logger,
	)

	return
}

// initializeExchangeRates fetches and caches exchange rates during application startup
func initializeExchangeRates(
	exchangeRateProvider provider.ExchangeRate,
	registryProvider registry.Provider,
	logger *slog.Logger,
) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the exchange cache with the provided registry provider
	exchangeCache := caching.NewExchangeCache(
		registryProvider,
		logger,
		15*time.Minute, // Default cache TTL
	)

	// Fetch rates from the provider
	rates, err := exchangeRateProvider.GetRates(ctx, "USD")
	if err != nil {
		return fmt.Errorf("failed to fetch exchange rates: %w", err)
	}

	// Cache the rates using ExchangeCache
	if err := exchangeCache.CacheRates(ctx, rates, exchangeRateProvider.Name()); err != nil {
		return fmt.Errorf("failed to cache exchange rates: %w", err)
	}

	logger.Info("Successfully fetched and cached exchange rates",
		"provider", exchangeRateProvider.Name(),
		"rates_count", len(rates),
	)

	return nil
}
