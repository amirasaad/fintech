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
	deps.RegistryProvider, err = GetDefaultRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize main registry provider: %w", err)
	}

	// Initialize currency registry
	deps.CurrencyRegistry, err = GetDefaultRegistry(cfg, logger)
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
		// Load currency metadata from embedded CSV
		logger.Info("Loading embedded currency metadata")
		entities, err := currencyfixtures.LoadCurrencyMetaCSV("")
		if err != nil {
			logger.Warn("Failed to load currency meta from CSV", "error", err)
		} else {
			logger.Info("Loading currency meta from fixture",
				"existing_count", count,
				"to_register", len(entities))
			for _, entity := range entities {
				if err := deps.CurrencyRegistry.Register(ctx, entity); err != nil {
					logger.Error("Failed to register currency", "code", entity.ID(), "error", err)
					// Continue with other currencies even if one fails
				}
			}
			logger.Info("Successfully loaded currency fixtures", "registered_count", len(entities))
		}
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
	exchangeProvider := infra_provider.NewExchangeRateAPIProvider(
		cfg.ExchangeRateAPIProviders.ExchangeRateApi,
		logger,
	)
	if err := initializeExchangeRates(
		exchangeProvider,
		deps.ExchangeRateRegistry,
		cfg.ExchangeRateCache,
		logger,
	); err != nil {
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

	// Initialize payment provider with the checkout registry
	deps.PaymentProvider = infra_provider.NewStripePaymentProvider(
		bus,
		deps.CheckoutRegistry, // Use the checkout-specific registry
		cfg.PaymentProviders.Stripe,
		logger,
	)

	return
}

// initializeExchangeRates fetches and caches exchange rates during application startup
func initializeExchangeRates(
	exchangeRateProvider provider.ExchangeRate,
	registryProvider registry.Provider,
	cfg *config.ExchangeRateCache,
	logger *slog.Logger,
) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the exchange cache with the provided registry provider
	exchangeCache := caching.NewExchangeCache(
		registryProvider,
		logger,
		cfg.TTL,
	)

	// Fetch rates from the provider
	rates, err := exchangeRateProvider.GetRates(ctx, "USD")
	if err != nil {
		return fmt.Errorf("failed to fetch exchange rates: %w", err)
	}

	// Cache the rates using ExchangeCache
	if err := exchangeCache.CacheRates(
		ctx,
		rates,
		exchangeRateProvider.Name(),
	); err != nil {
		return fmt.Errorf("failed to cache exchange rates: %w", err)
	}

	logger.Info("Successfully fetched and cached exchange rates",
		"provider", exchangeRateProvider.Name(),
		"rates_count", len(rates),
	)

	return nil
}
