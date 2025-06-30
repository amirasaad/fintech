package handler

import (
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

func AccountRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error)) {
	app.Post("/account", func(c *fiber.Ctx) error {
		log.Infof("Creating new account")
		service := service.NewAccountService(uowFactory)
		a, err := service.CreateAccount()
		if err != nil {
			log.Errorf("Failed to create account: %v", err)
			status := service.ErrorToStatusCode(err)
			return c.Status(status).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		log.Infof("Account created: %+v", a)
		return c.JSON(a)
	})

	app.Post("/account/:id/deposit", func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		type DepositRequest struct {
			Amount float64 `json:"amount" xml:"amount" form:"amount"`
		}
		var request DepositRequest
		err = c.BodyParser(&request)
		if err != nil {
			log.Errorf("Failed to parse deposit request: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		tx, err := service.Deposit(id, request.Amount)
		if err != nil {
			log.Errorf("Failed to deposit: %v", err)
			status := service.ErrorToStatusCode(err)
			return c.Status(status).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(tx)
	})

	app.Post("/account/:id/withdraw", func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for withdrawal: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		type WithdrawRequest struct {
			Amount float64 `json:"amount"`
		}
		var request WithdrawRequest
		err = c.BodyParser(&request)
		if err != nil {
			log.Errorf("Failed to parse withdrawal request: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		tx, err := service.Withdraw(id, request.Amount)
		if err != nil {
			log.Errorf("Failed to withdraw: %v", err)
			status := service.ErrorToStatusCode(err)
			return c.Status(status).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(tx)
	})

	app.Get("/account/:id/transactions", func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for transactions: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		tx, err := service.GetTransactions(id)
		if err != nil {
			log.Errorf("Failed to list transactions for account ID %s: %v", id, err)
			status := service.ErrorToStatusCode(err)
			return c.Status(status).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(tx)
	})

	app.Get("/account/:id/balance", func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for balance: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		balance, err := service.GetBalance(id)
		if err != nil {
			log.Errorf("Failed to fetch balance for account ID %s: %v", id, err)
			status := service.ErrorToStatusCode(err)
			return c.Status(status).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"balance": balance,
		})
	})
}
