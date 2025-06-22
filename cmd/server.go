package main

import (
	"github.com/amirasaad/fintech/internal/infra"
	"github.com/amirasaad/fintech/pkg/handler"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	uowFactory := func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("App is working! 🚀")
	})

	handler.AccountRoutes(app, uowFactory)

	err := app.Listen(":3000")
	if err != nil {
		panic("App is not starting..")
	}
}
