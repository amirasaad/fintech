package handler

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/fatih/color"

	"github.com/amirasaad/fintech/infra"

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
		_, _ = color.New(color.FgRed).Fprintln(os.Stderr, "Failed to load application configuration:", err)
		panic("Failed to load application configuration")
	}

	appEnv := os.Getenv("APP_ENV")
	app := webapi.NewApp(
		service.NewAccountService(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(cfg.DB, appEnv)
		}, domain.NewStubCurrencyConverter()),
		service.NewUserService(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(cfg.DB, appEnv)
		}),
		service.NewAuthService(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(cfg.DB, appEnv)
		}, service.NewJWTAuthStrategy(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(cfg.DB, appEnv)
		}, cfg.Jwt)),
		*cfg,
	)
	return adaptor.FiberApp(app)
}
