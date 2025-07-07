package handler

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"net/http"

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
	db, err := infra.NewDBConnection()
	if err != nil {
		panic(err)
	}
	app := webapi.NewApp(
		service.NewAccountService(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(db)
		}, domain.NewStubCurrencyConverter()),
		service.NewUserService(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(db)
		}),
		service.NewAuthService(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(db)
		}, service.NewJWTAuthStrategy(func() (repository.UnitOfWork, error) {
			return infra.NewGormUoW(db)
		})),
	)
	return adaptor.FiberApp(app)
}
