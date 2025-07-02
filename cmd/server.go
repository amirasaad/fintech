package main

import (
	"log"

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
	webapi.UserRoutes(app, func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	})
	webapi.AuthRoutes(app, func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW()
	})

	// JWT Middleware
	// app.Use(middleware.Protected())

	log.Fatal(app.Listen(":3000"))
}
