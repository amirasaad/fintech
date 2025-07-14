package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyservice "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/amirasaad/fintech/pkg/service/user"
	"github.com/amirasaad/fintech/webapi"
)

// @title Fintech API
// @version 1.0.0
// @description Fintech API documentation
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/MIT
// @host fintech-beryl-beta.vercel.app
// @BasePath /
func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(log.Writer(), &slog.HandlerOptions{Level: slog.LevelDebug}))
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

	// Create services
	accountSvc := account.NewService(account.ServiceDeps{
		Uow:             uow,
		Converter:       currencyConverter,
		Logger:          logger,
		PaymentProvider: provider.NewMockPaymentProvider(),
	})
	userSvc := user.NewUserService(uow, logger)
	authStrategy := auth.NewJWTAuthStrategy(uow, cfg.Jwt, logger)
	authSvc := auth.NewAuthService(uow, authStrategy, logger)
	currencySvc := currencyservice.NewCurrencyService(currencyRegistry, logger)

	logger.Info("Starting fintech server", "port", ":3000")
	log.Fatal(webapi.NewApp(
		accountSvc,
		userSvc,
		authSvc,
		currencySvc,
		cfg,
	).Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)))
}
