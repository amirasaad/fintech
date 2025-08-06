package handler

import (
	"context"
	"github.com/amirasaad/fintech/pkg/config"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/amirasaad/fintech/webapi"

	"github.com/amirasaad/fintech/infra/eventbus"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/registry"

	infra_repository "github.com/amirasaad/fintech/infra/repository"

	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// Handler is the main entry point of the application.
// Think of it like the main() method
func Handler(w http.ResponseWriter, r *http.Request) {
	// This is needed to set the proper request path in `*fiber.Ctx`
	r.RequestURI = r.URL.String()

	handler().ServeHTTP(w, r)
}

// building the fiber application
func handler() http.HandlerFunc {
	logHandler := slog.NewTextHandler(log.Writer(), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		log.Fatal(err)
	}
	ctx := context.Background()
	// Initialize the global currency registry with Redis if configured
	if cfg.Redis.URL != "" {
		// Initialize with Redis cache and key prefix
		// Initialize the global currency registry with Redis cache
		redisURL := cfg.Redis.URL
		keyPrefix := cfg.Redis.KeyPrefix + "currency:"
		if err = currency.InitializeGlobalRegistry(ctx, redisURL, keyPrefix); err != nil {
			logger.Error("Failed to initialize global currency registry with Redis",
				"error", err,
				"redis_url", cfg.Redis.URL,
				"key_prefix", cfg.Redis.KeyPrefix)
			log.Fatal(err)
		}
	} else {
		// Initialize with in-memory cache
		if err = currency.InitializeGlobalRegistry(ctx); err != nil {
			logger.Error(
				"Failed to initialize global currency registry",
				"error", err,
				"redis_configured", cfg.Redis.URL != "")
			log.Fatal(err)
		}
	}
	currencyRegistry := currency.GetGlobalRegistry()

	// Log currency registry initialization details
	if cfg.Redis.URL != "" {
		logger.Info(
			"Currency registry initialized with Redis cache",
			"redis_url", cfg.Redis.URL,
			"key_prefix", cfg.Redis.KeyPrefix)
	} else {
		logger.Info(
			"Currency registry initialized with in-memory cache",
			"redis_configured", cfg.Redis.URL != "")
	}

	// Create a Redis-backed registry for the checkout service
	checkoutRegistry, err := registry.NewBuilder().
		WithName("checkout").
		WithRedis(cfg.Redis.URL).
		WithKeyPrefix(cfg.Redis.KeyPrefix+"checkout:").
		WithCache(1000, 15*time.Minute). // Cache up to 1000 items for 15 minutes
		BuildRegistry()
	if err != nil {
		logger.Error(
			"Failed to create checkout registry",
			"error", err,
			"redis_configured", cfg.Redis.URL != "")
		log.Fatal(err)
	}

	if cfg.Redis.URL != "" {
		logger.Info(
			"Checkout registry initialized with Redis cache",
			"redis_url", cfg.Redis.URL,
			"key_prefix", cfg.Redis.KeyPrefix)
	} else {
		logger.Info(
			"Checkout registry initialized with in-memory cache",
			"redis_configured", cfg.Redis.URL != "")
	}

	// Initialize DB connection ONCE
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error(
			"Failed to initialize database",
			"error", err,
			"database_url_configured", cfg.DB.Url != "")
		log.Fatal(err)
	}

	// Create UOW factory using the shared db
	uow := infra_repository.NewUoW(db)

	// Create exchange rate system
	currencyConverter, err := infra.NewExchangeRateSystem(logger, cfg.Exchange)
	if err != nil {
		logger.Error(
			"Failed to initialize exchange rate system",
			"error", err,
			"exchange_rate_api_configured", cfg.Exchange.ApiKey != "")
		log.Fatal(err)
	}

	// Define event types for Redis event bus

	bus, err := eventbus.NewWithRedis(
		cfg.Redis.URL,
		logger,
	)
	// bus := eventbus.NewWithMemory(logger)
	if err != nil {
		logger.Error(
			"Failed to initialize event bus",
			"error", err,
			"redis_configured", cfg.Redis.URL != "")
		log.Fatal(err)
	}

	logger.Info(
		"Starting fintech server",
		"port", ":3000",
		"redis_configured", cfg.Redis.URL != "")
	a := webapi.SetupApp(config.Deps{
		Uow:               uow,
		EventBus:          bus,
		CurrencyConverter: currencyConverter,
		CurrencyRegistry:  currencyRegistry,
		PaymentProvider: provider.NewStripePaymentProvider(
			bus,
			&cfg.PaymentProviders.Stripe,
			checkout.NewService(checkoutRegistry),
			logger),
		Logger: logger,
		Config: cfg,
	})
	return adaptor.FiberApp(a)
}
