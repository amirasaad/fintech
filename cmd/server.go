package main

import (
	"github.com/amirasaad/fintech/internal/infra"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/webapi"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("App is working! 🚀")
	})

	webapi.AccountRoutes(app, func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	})

	err := app.Listen(":3000")
	if err != nil {
		panic("App is not starting..")
	}
}
