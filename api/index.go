package handler

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"

	infra_repository "github.com/amirasaad/fintech/infra/repository"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/amirasaad/fintech/webapi"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// Handler is the main entry point of the application. Think of it like the main() method
func Handler(w http.ResponseWriter, r *http.Request) {
	// This is needed to set the proper request path in `*fiber.Ctx`
	r.RequestURI = r.URL.String()

	handler().ServeHTTP(w, r)
}

// building the fiber application
func handler() http.HandlerFunc {
	logger := slog.New(slog.NewTextHandler(log.Writer(), &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		log.Fatal(err)
	}
	currencyConverter, err := infra.NewExchangeRateSystem(logger, *cfg)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system", "error", err)
		log.Fatal(err)
	}
	// Initialize currency registry
	ctx := context.Background()
	currencyRegistry, err := currency.NewCurrencyRegistry(ctx)
	if err != nil {
		logger.Error("Failed to initialize currency registry", "error", err)
		log.Fatal(err)
	}
	logger.Info("Currency registry initialized successfully")
	currencySvc := service.NewCurrencyService(currencyRegistry, logger)

	// Create UOW factory
	uowFactory := func() (repository.UnitOfWork, error) {
		return infra_repository.NewGormUoW(cfg.DB, cfg.Env)
	}

	app := webapi.NewApp(
		service.NewAccountService(uowFactory, currencyConverter, logger),
		service.NewUserService(uowFactory, logger),
		service.NewAuthService(uowFactory,
			service.NewJWTAuthStrategy(
				uowFactory, cfg.Jwt, logger,
			), logger),
		currencySvc,
		cfg,
	)
	return adaptor.FiberApp(app)
}
