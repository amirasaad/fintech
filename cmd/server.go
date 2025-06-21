package main

import (
	"github.com/amirasaad/fintech/internal/handler"
	"github.com/amirasaad/fintech/internal/infra"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	infra.Connect()
	db := infra.DB
	accountRepo := infra.NewAccountRepository(db)
	transactionRepo := infra.NewTransactionRepository(db)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("App is working! 🚀")
	})

	handler.AccountRoutes(app, accountRepo, transactionRepo)

	app.Listen(":3000")
}
