package handler

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/pkg/config"

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
	logger := slog.New(slog.NewTextHandler(log.Writer(), nil))
	slog.SetDefault(logger)
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		log.Fatal(err)
	}
	currencyConverter, err := infra.NewExchangeRateSystem(slog.Default(), cfg.Exchange)
	if err != nil {
		logger.Error("Failed to initialize exchange rate system", "error", err)
		log.Fatal(err)
	}

	appEnv := os.Getenv("APP_ENV")
	uow := func() (repository.UnitOfWork, error) {
		return infra_repository.NewGormUoW(cfg.DB, appEnv)
	}
	app := webapi.NewApp(
		service.NewAccountService(uow, currencyConverter),
		service.NewUserService(uow),
		service.NewAuthService(uow, service.NewJWTAuthStrategy(uow, cfg.Jwt)),
		cfg,
	)
	return adaptor.FiberApp(app)
}
