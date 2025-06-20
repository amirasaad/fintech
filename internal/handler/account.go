package handler

import (
	"github.com/amirasaad/fintech/internal/domain"
	"github.com/amirasaad/fintech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

func AccountRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error)) {
	app.Post("/account", func(c *fiber.Ctx) error {
		log.Infof("Creating new account")
		uow, err := uowFactory()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = uow.Begin()

		a := domain.NewAccount()
		err = uow.AccountRepository().Create(a)
		if err != nil {
			_ = uow.Rollback()
			log.Errorf("Failed to create account: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Account created: %+v", a)
		err = uow.Commit()
		if err != nil {
			return err
		}
		return c.JSON(a)
	})

	app.Post("/account/:id/deposit", func(c *fiber.Ctx) error {
		uow, err := uowFactory()
		if err != nil {
			return err
		}
		err = uow.Begin()
		if err != nil {
			return err
		}
		type DepositRequest struct {
			Amount float64 `json:"amount" xml:"amount" form:"amount"`
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			_ = uow.Rollback()
			log.Errorf("Invalid account ID for deposit: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Fetching account for id %s", id)
		a, err := uow.AccountRepository().Get(id)
		log.Infof("Fetced Account %+v, err %s", a, err)
		if err != nil {
			err = uow.Rollback()
			if err != nil {
				return err
			}
			log.Errorf("Failed to fetch account for id %s", id)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Account not found",
			})
		}
		var request DepositRequest
		err = c.BodyParser(&request)
		log.Infof("Account deposit %+v", request)
		if err != nil {
			_ = uow.Rollback()
			log.Errorf("Failed to parse account deposit %+v with error %s", request, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Depositing amount %f", request.Amount)
		tx, err := a.Deposit(request.Amount)
		if err != nil {
			_ = uow.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Depositing transaction amount %+v", tx)
		err = uow.AccountRepository().Update(a)
		if err != nil {
			_ = uow.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		err = uow.TransactionRepository().Create(tx)
		if err != nil {
			_ = uow.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Deposit successful for account %s, transaction: %+v", id, tx)
		err = uow.Commit()
		if err != nil {
			return err
		}
		return c.JSON(tx)
	})

	app.Post("/account/:id/withdraw", func(c *fiber.Ctx) error {
		uow, err := uowFactory()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		err = uow.Begin()
		if err != nil {
			return err
		}
		accountRepo := uow.AccountRepository()
		transactionRepo := uow.TransactionRepository()
		type WithdrawRequest struct {
			Amount float64 `json:"amount"`
		}
		var request WithdrawRequest
		err = c.BodyParser(&request)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Withdraw request: %+v", request)

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for withdrawal: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		log.Infof("Fetching account for withdrawal for id %s", id)
		a, err := accountRepo.Get(id)
		if err != nil {
			log.Errorf("Failed to fetch account for withdrawal for id %s: %v", id, err)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Account not found",
			})
		}

		log.Infof("Processing withdrawal for account %s, amount %f", id, request.Amount)
		tx, err := a.Withdraw(request.Amount)
		if err != nil {
			_ = uow.Rollback()
			log.Errorf("Failed to process withdrawal for account %s, amount %f: %v", id, request.Amount, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		err = transactionRepo.Create(tx)
		if err != nil {
			_ = uow.Rollback()
			log.Errorf("Failed to create transaction record for withdrawal: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		err = accountRepo.Update(a)
		if err != nil {
			_ = uow.Rollback()
			log.Errorf("Failed to update account balance after withdrawal: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Withdrawal successful for account %s, transaction: %+v", id, tx)
		err = uow.Commit()
		if err != nil {
			return err
		}
		return c.JSON(tx)
	})

	app.Get("/account/:id/transactions", func(c *fiber.Ctx) error {
		uow, err := uowFactory()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Fetching transactions for account ID: %s", c.Params("id"))
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for transactions: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		tx, err := uow.TransactionRepository().List(id)
		if err != nil {
			log.Errorf("Failed to list transactions for account ID %s: %v", id, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Successfully fetched %d transactions for account ID %s", len(tx), id)
		return c.JSON(tx)
	})

	app.Get("/account/:id/balance", func(c *fiber.Ctx) error {
		uow, err := uowFactory()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		log.Infof("Fetching balance for account ID: %s", c.Params("id"))
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for balance: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		a, err := uow.AccountRepository().Get(id)
		if err != nil {
			log.Errorf("Failed to fetch account for balance for id %s: %v", id, err)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Account not found",
			})
		}
		log.Infof("Successfully fetched balance for account ID %s: %f", id, a.GetBalance())
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"balance": a.GetBalance(),
		})
	})
}
