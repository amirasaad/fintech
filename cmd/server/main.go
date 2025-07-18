package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amirasaad/fintech/app"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/eventbus"

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
	handler := log.NewWithOptions(os.Stdout, log.Options{
		ReportTimestamp: true,
		TimeFunction:    log.NowUTC,
		TimeFormat:      time.Kitchen,
		ReportCaller:    true,
		Prefix:          "Server 🗄️ ",
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
	currencyRegistry, err := currency.NewCurrencyRegistry(ctx)
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
	currencyConverter, err := infra.NewExchangeRateSystem(logger, *cfg)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system", "error", err)
		log.Fatal(err)
	}

	logger.Info("Starting fintech server", "port", ":3000")
	log.Fatal(app.New(config.Deps{
		Uow:               uow,
		CurrencyConverter: currencyConverter,
		CurrencyRegistry:  currencyRegistry,
		Logger:            logger,
		PaymentProvider:   provider.NewStripePaymentProvider(cfg.PaymentProviders.Stripe.ApiKey, logger),
		EventBus:          eventbus.NewSimpleEventBus(),
		Config:            cfg,
	}).Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)))
}
