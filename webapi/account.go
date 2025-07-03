// The `package handler` in this code snippet is defining a Go package that contains HTTP handler
// functions for handling various account-related operations such as creating an account, depositing
// funds, withdrawing funds, retrieving transactions, and checking the account balance. These handler
// functions are responsible for processing incoming HTTP requests, interacting with the `service`
// layer to perform the necessary business logic, and returning appropriate responses to the client.
// The handlers are using the Fiber web framework for building the HTTP server and handling routing.
package webapi

import (
	"github.com/amirasaad/fintech/middleware"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

func AccountRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error)) {
	app.Post("/account", middleware.Protected(), CreateAccount(uowFactory))
	app.Post("/account/:id/deposit", middleware.Protected(), Deposit(uowFactory))
	app.Post("/account/:id/withdraw", middleware.Protected(), Withdraw(uowFactory))
	app.Get("/account/:id/balance", middleware.Protected(), GetBalance(uowFactory))
	app.Get("/account/:id/transactions", middleware.Protected(), GetTransactions(uowFactory))
}

// The `AccountRoutes` function defines various HTTP routes for account-related operations using Fiber
// in Go.
func CreateAccount(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Infof("Creating new account")
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		service := service.NewAccountService(uowFactory)
		a, err := service.CreateAccount(userID)
		if err != nil {
			log.Errorf("Failed to create account: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to create account", err.Error())
		}

		log.Infof("Account created: %+v", a)
		return c.JSON(a)
	}
}
func Deposit(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}
		type DepositRequest struct {
			Amount float64 `json:"amount" xml:"amount" form:"amount"`
		}
		var request DepositRequest
		err = c.BodyParser(&request)
		if err != nil {
			log.Errorf("Failed to parse deposit request: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Failed to parse deposit request", err.Error())
		}

		tx, err := service.Deposit(userID, id, request.Amount)
		if err != nil {
			log.Errorf("Failed to deposit: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to deposit", err.Error())
		}
		return c.JSON(tx)
	}
}

func Withdraw(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for withdrawal: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}
		type WithdrawRequest struct {
			Amount float64 `json:"amount"`
		}
		var request WithdrawRequest
		err = c.BodyParser(&request)
		if err != nil {
			log.Errorf("Failed to parse withdrawal request: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Failed to parse withdrawal request", err.Error())
		}
		tx, err := service.Withdraw(userID, id, request.Amount)
		if err != nil {
			log.Errorf("Failed to withdraw: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to withdraw", err.Error())
		}
		return c.JSON(tx)
	}
}

func GetTransactions(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for transactions: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}

		tx, err := service.GetTransactions(userID, id)
		if err != nil {
			log.Errorf("Failed to list transactions for account ID %s: %v", id, err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to list transactions", err.Error())
		}
		return c.JSON(tx)
	}
}

func GetBalance(uowFactory func() (repository.UnitOfWork, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		service := service.NewAccountService(uowFactory)
		userID, err := GetCurrentUserId(c)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "invalid user ID", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for balance: %v", err)
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}

		balance, err := service.GetBalance(userID, id)
		if err != nil {
			log.Errorf("Failed to fetch balance for account ID %s: %v", id, err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to fetch balance", err.Error())
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"balance": balance,
		})
	}
}
