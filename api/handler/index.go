package handler

import (
	"github.com/amirasaad/fintech/webapi/infra"
	"net/http"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Handler is the main entry point of the application. Think of it like the main() method
func Handler(w http.ResponseWriter, r *http.Request) {
	// This is needed to set the proper request path in `*fiber.Ctx`
	r.RequestURI = r.URL.String()

	handler().ServeHTTP(w, r)
}

// building the fiber application
func handler() http.HandlerFunc {
	app := fiber.New()

	app.Use(limiter.New())
	app.Use(recover.New())

	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"uri":  ctx.Request().URI().String(),
			"path": ctx.Path(),
		})
	})

	webapi.AuthRoutes(app, func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	})

	webapi.UserRoutes(app, func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	})

	webapi.AccountRoutes(app, func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	})

	return adaptor.FiberApp(app)
}
