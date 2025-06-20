package handler

import (
	"github.com/amirasaad/fintech/internal/account"
	"github.com/amirasaad/fintech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

func AccountRoutes(app *fiber.App, accountRepo repository.AccountRepository, transactionRepo repository.TransactionRepository) {
	app.Post("/account", func(c *fiber.Ctx) error {
		a := account.New()
		err := accountRepo.Create(a)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(a)
	})

	app.Post("/account/:id/deposit", func(c *fiber.Ctx) error {
		type DepositRequest struct {
			Amount float64 `json:"amount" xml:"amount" form:"amount"`
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Fetching account for id %s", id)
		a, err := accountRepo.Get(id)
		log.Infof("Fetced Account %+v, err %s", a, err)
		if err != nil {
			log.Errorf("Failed to fetch account for id %s", id)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Account not found",
			})
		}
		var request DepositRequest
		err = c.BodyParser(&request)
		log.Infof("Account deposit %+v", request)
		if err != nil {
			log.Errorf("Failed to parse account deposit %+v with error %s", request, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Depositing amount %f", request.Amount)
		transaction, err := a.Deposit(request.Amount)
		log.Infof("Depositing transaction amount %+v", transaction)
		err = accountRepo.Update(a)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		err = transactionRepo.Create(transaction)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(transaction)
	})

	app.Post("/account/:id/withdraw", func(c *fiber.Ctx) error {
		type WithdrawRequest struct {
			Amount float64 `json:"amount"`
		}
		var request WithdrawRequest
		err := c.BodyParser(&request)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		a, err := accountRepo.Get(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		transaction, err := a.Withdraw(request.Amount)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		err = transactionRepo.Create(transaction)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		err = accountRepo.Update(a)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(transaction)
	})

	app.Get("/account/:id/transactions", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		tx, err := transactionRepo.List(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(tx)
	})

	app.Get("/account/:id/balance", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		a, err := accountRepo.Get(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"balance": a.GetBalance(),
		})
	})
}
