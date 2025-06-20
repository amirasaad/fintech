package main

import (
	"github.com/amirasaad/fintech/internal/database"
	"github.com/amirasaad/fintech/internal/handler"
	"github.com/amirasaad/fintech/internal/repository"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	database.Connect()
	db := database.DB
	accountRepo := repository.NewAccountRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	handler.AccountRoutes(app, accountRepo, transactionRepo)

	app.Listen(":3000")
}
