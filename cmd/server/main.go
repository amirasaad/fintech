package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amirasaad/fintech/webapi"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/charmbracelet/log"
)

// @title Fintech API
// @version 1.0.0
// @description Fintech API documentation
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/MIT
// @host localhost:3000
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Enter your Bearer token in the format: `Bearer {token}`"
func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(a.Key, a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	})
	// Setup structured logging
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Load application configuration
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		log.Fatal(err)
	}

	logger.Info("Configuration loaded successfully",
		"database_url_configured", cfg.DB.Url != "",
		"jwt_expiry", cfg.Jwt.Expiry,
		"exchange_rate_api_configured", cfg.Exchange.ApiKey != "")

	// Initialize currency registry
	ctx := context.Background()
	currencyRegistry, err := currency.NewRegistry(ctx)
	if err != nil {
		logger.Error("Failed to initialize currency registry", "error", err)
		log.Fatal(err)
	}
	logger.Info("Currency registry initialized successfully")

	// Initialize DB connection ONCE
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		log.Fatal(err)
	}

	// Create UOW factory using the shared db
	uow := infra_repository.NewUoW(db)

	// Create exchange rate system
	currencyConverter, err := infra.NewExchangeRateSystem(logger, cfg.Exchange)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system", "error", err)
		log.Fatal(err)
	}

	// Define event types for Redis event bus

	bus, err := eventbus.NewWithRedis(
		cfg.Redis.URL,
		logger,
	)
	// bus := eventbus.NewWithMemory(logger)
	if err != nil {
		logger.Error("Failed to initialize event bus", "error", err)
		log.Fatal(err)
	}

	logger.Info("Starting fintech server", "port", ":3000")
	log.Fatal(webapi.SetupApp(config.Deps{
		Uow:               uow,
		EventBus:          bus,
		CurrencyConverter: currencyConverter,
		CurrencyRegistry:  currencyRegistry,
		PaymentProvider: provider.NewStripePaymentProvider(
			&cfg.PaymentProviders.Stripe,
			logger,
		),
		Config: cfg,
		Logger: logger,
	}).Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)))
}
