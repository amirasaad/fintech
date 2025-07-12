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

	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyservice "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/amirasaad/fintech/pkg/service/user"
	"github.com/amirasaad/fintech/webapi/common"
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
	currencySvc := currencyservice.NewCurrencyService(currencyRegistry, logger)

	// Initialize DB connection ONCE
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		log.Fatal(err)
	}

	// Create UOW using the shared db
	uow := infra_repository.NewUoW(db)

	app := common.NewApp(
		account.NewAccountService(uow, currencyConverter, logger),
		user.NewUserService(uow, logger),
		auth.NewAuthService(uow,
			auth.NewJWTAuthStrategy(
				uow, cfg.Jwt, logger,
			), logger),
		currencySvc,
		cfg,
	)
	return adaptor.FiberApp(app)
}
