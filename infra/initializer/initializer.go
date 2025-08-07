package initializer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/infra"
	infra_eventbus "github.com/amirasaad/fintech/infra/eventbus"
	infra_provider "github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/repository"
)

// Deps contains all the dependencies needed by the application
type Deps struct {
	Uow              repository.UnitOfWork
	EventBus         eventbus.Bus
	CurrencyRegistry *currency.Registry
	PaymentProvider  provider.PaymentProvider
	Logger           *slog.Logger
}

// InitializeDependencies initializes all the application dependencies
func InitializeDependencies(logger *slog.Logger) (*app.Deps, *config.AppConfig, error) {
	// Load configuration
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		return nil, nil, err
	}

	// Initialize currency registry
	currencyRegistry, err := initCurrencyRegistry(cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	// Initialize checkout registry
	checkoutRegistry, err := initCheckoutRegistry(cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	// Initialize database
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		return nil, nil, err
	}

	// Initialize unit of work
	uow := infra_repository.NewUoW(db)

	// Initialize event bus
	var bus eventbus.Bus
	if cfg.Redis.URL != "" {
		redisBus, err := infra_eventbus.NewWithRedis(cfg.Redis.URL, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize Redis event bus: %w", err)
		}
		bus = redisBus
	} else {
		// Fall back to in-memory event bus if Redis is not configured
		bus = infra_eventbus.NewWithMemory(logger)
	}

	// Initialize payment provider
	paymentProvider := infra_provider.NewStripePaymentProvider(
		bus,
		&cfg.PaymentProviders.Stripe,
		checkout.NewService(checkoutRegistry),
		logger,
	)

	return &app.Deps{
		Uow:              uow,
		EventBus:         bus,
		CurrencyRegistry: currencyRegistry,
		PaymentProvider:  paymentProvider,
		Logger:           logger,
	}, cfg, nil
}

func initCurrencyRegistry(cfg *config.AppConfig, logger *slog.Logger) (*currency.Registry, error) {
	ctx := context.Background()
	if cfg.Redis.URL != "" {
		keyPrefix := cfg.Redis.KeyPrefix + "currency:"
		if err := currency.InitializeGlobalRegistry(ctx, cfg.Redis.URL, keyPrefix); err != nil {
			logger.Error("Failed to initialize global currency registry with Redis",
				"error", err,
				"redis_url", cfg.Redis.URL,
				"key_prefix", keyPrefix)
			return nil, err
		}
		logger.Info("Currency registry initialized with Redis cache",
			"redis_url", cfg.Redis.URL,
			"key_prefix", keyPrefix)
	} else {
		if err := currency.InitializeGlobalRegistry(ctx); err != nil {
			logger.Error("Failed to initialize global currency registry", "error", err)
			return nil, err
		}
		logger.Info("Currency registry initialized with in-memory cache")
	}
	return currency.GetGlobalRegistry(), nil
}

func initCheckoutRegistry(cfg *config.AppConfig, logger *slog.Logger) (registry.Provider, error) {
	checkoutRegistry, err := registry.NewBuilder().
		WithName("checkout").
		WithRedis(cfg.Redis.URL).
		WithKeyPrefix(cfg.Redis.KeyPrefix+"checkout:").
		WithCache(1000, 15*time.Minute).
		BuildRegistry()
	if err != nil {
		logger.Error("Failed to create checkout registry",
			"error", err,
			"redis_configured", cfg.Redis.URL != "")
		return nil, err
	}
	return checkoutRegistry, nil
}
