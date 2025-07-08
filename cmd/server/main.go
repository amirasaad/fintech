package main

import (
	"log"
	"log/slog"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
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
	logger := slog.New(slog.NewTextHandler(log.Writer(), nil))
	slog.SetDefault(logger)

	// Load application configuration
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		log.Fatal(err)
	}

	logger.Info("Configuration loaded successfully",
		"database_url_configured", cfg.DB.Url != "",
		"jwt_expiry", cfg.Auth.JwtExpiry,
		"exchange_rate_api_configured", cfg.Exchange.ApiKey != "")

	// Create UOW factory
	uowFactory := func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(cfg.DB)
	}

	// Create exchange rate system
	currencyConverter, err := infra.NewExchangeRateSystem(logger, cfg.Exchange)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system", "error", err)
		log.Fatal(err)
	}

	// Create services
	accountSvc := service.NewAccountService(uowFactory, currencyConverter)
	userSvc := service.NewUserService(uowFactory)
	authSvc := service.NewAuthService(uowFactory, service.NewJWTAuthStrategy(uowFactory))

	logger.Info("Starting fintech server", "port", ":3000")
	log.Fatal(webapi.NewApp(accountSvc, userSvc, authSvc).Listen(":3000"))
}
