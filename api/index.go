package handler

import (
	"context"
	"github.com/amirasaad/fintech/webapi"
	"log"
	"log/slog"
	"net/http"

	"github.com/amirasaad/fintech/infra/eventbus"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/pkg/currency"

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
	currencyConverter, err := infra.NewExchangeRateSystem(logger, cfg.Exchange)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system", "error", err)
		log.Fatal(err)
	}
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

	// Create UOW using the shared db
	uow := infra_repository.NewUoW(db)

	a := webapi.SetupApp(config.Deps{
		Uow:               uow,
		EventBus:          eventbus.NewWithMemoryAsync(logger),
		CurrencyConverter: currencyConverter,
		CurrencyRegistry:  currencyRegistry,
		PaymentProvider:   provider.NewMockPaymentProvider(),
		Logger:            logger,
		Config:            cfg,
	})
	return adaptor.FiberApp(a)
}
