package handler

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"

	infra_repository "github.com/amirasaad/fintech/infra/repository"

	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyservice "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/amirasaad/fintech/pkg/service/user"
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
	// Configure logger for Vercel - use stderr for better visibility
	var logLevel slog.Level
	if os.Getenv("VERCEL_ENV") == "production" {
		logLevel = slog.LevelInfo
	} else {
		logLevel = slog.LevelDebug
	}

	// Use stderr for Vercel logging visibility
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	// Log application startup for Vercel visibility
	logger.Info("Application starting",
		"environment", os.Getenv("VERCEL_ENV"),
		"function", "api/index.go",
		"started_at", time.Now(),
	)

	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration",
			"error", err,
			"function", "api/index.go",
		)
		log.Fatal(err)
	}

	currencyConverter, err := infra.NewExchangeRateSystem(logger, *cfg)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system",
			"error", err,
			"function", "api/index.go",
		)
		log.Fatal(err)
	}

	// Initialize currency registry
	ctx := context.Background()
	currencyRegistry, err := currency.NewCurrencyRegistry(ctx)
	if err != nil {
		logger.Error("Failed to initialize currency registry",
			"error", err,
			"function", "api/index.go",
		)
		log.Fatal(err)
	}
	logger.Info("Currency registry initialized successfully",
		"function", "api/index.go",
		"registry_ready", true,
	)

	currencySvc := currencyservice.NewCurrencyService(currencyRegistry, logger)

	// Initialize DB connection ONCE
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error("Failed to initialize database",
			"error", err,
			"function", "api/index.go",
		)
		log.Fatal(err)
	}

	// Create UOW using the shared db
	uow := infra_repository.NewUoW(db)

	app := webapi.NewApp(
		account.NewAccountService(uow, currencyConverter, logger),
		user.NewUserService(uow, logger),
		auth.NewAuthService(uow,
			auth.NewJWTAuthStrategy(
				uow, cfg.Jwt, logger,
			), logger),
		currencySvc,
		cfg,
	)

	logger.Info("Application initialized successfully",
		"function", "api/index.go",
		"services_ready", true,
	)

	return adaptor.FiberApp(app)
}
