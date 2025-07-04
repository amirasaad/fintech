package webapi

import (
	"time"

	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func NewApp(uowFactory func() (repository.UnitOfWork, error)) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default to 500 if status code cannot be determined
			status := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			}
			return ErrorResponseJSON(c, status, "Internal Server Error", err.Error())
		},
	})

	app.Use(limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return ErrorResponseJSON(c, fiber.StatusTooManyRequests, "Too Many Requests", "Rate limit exceeded")
		},
	}))
	app.Use(recover.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("App is working! 🚀")
	})

	AccountRoutes(app, uowFactory)
	UserRoutes(app, uowFactory)
	AuthRoutes(app, uowFactory)

	return app
}
