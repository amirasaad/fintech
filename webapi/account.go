// AccountRoutes registers HTTP routes for account-related operations using the Fiber web framework.
// It sets up endpoints for creating accounts, depositing and withdrawing funds, retrieving account balances,
// and listing account transactions. All routes are protected by authentication middleware and require a valid user context.
//
// Parameters:
//   - app: The Fiber application instance to register routes on.
//   - uowFactory: A factory function that returns a new UnitOfWork for database operations.
//
// Routes:
//   - POST   /account                   : Create a new account for the authenticated user.
//   - POST   /account/:id/deposit       : Deposit funds into the specified account.
//   - POST   /account/:id/withdraw      : Withdraw funds from the specified account.
//   - GET    /account/:id/balance       : Retrieve the balance of the specified account.
//   - GET    /account/:id/transactions  : List transactions for the specified account.

package webapi

import (
	"github.com/amirasaad/fintech/pkg/middleware"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AccountRoutes(app *fiber.App, uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) {
	app.Post("/account", middleware.Protected(), CreateAccount(uowFactory, strategy))
	app.Post("/account/:id/deposit", middleware.Protected(), Deposit(uowFactory, strategy))
	app.Post("/account/:id/withdraw", middleware.Protected(), Withdraw(uowFactory, strategy))
	app.Get("/account/:id/balance", middleware.Protected(), GetBalance(uowFactory, strategy))
	app.Get("/account/:id/transactions", middleware.Protected(), GetTransactions(uowFactory, strategy))
}

// CreateAccount returns a Fiber handler for creating a new account for the current user.
// It extracts the user ID from the request context, initializes the account service using the provided
// UnitOfWork factory, and attempts to create a new account. On success, it returns the created account as JSON.
// On failure, it logs the error and returns an appropriate error response.
//
// Parameters:
//   - uowFactory: A function that returns a new instance of repository.UnitOfWork and an error.
//
// Returns:
//   - fiber.Handler: The HTTP handler function for account creation.
func CreateAccount(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Infof("Creating new account")
		authSvc := service.NewAuthService(uowFactory, strategy)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
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
		return c.Status(fiber.StatusCreated).JSON(Response{Status: fiber.StatusCreated, Message: "Account created", Data: a})
	}
}

// Deposit returns a Fiber handler for depositing an amount into a user's account.
// It expects a UnitOfWork factory function as a dependency for transactional operations.
// The handler parses the current user ID from the request context, validates the account ID from the URL,
// and parses the deposit amount from the request body. If successful, it performs the deposit operation
// using the AccountService and returns the transaction as JSON. On error, it logs the issue and returns
// an appropriate JSON error response.
//
//	@param uowFactory A function that returns a new repository.UnitOfWork and error.
//	@return fiber.Handler A Fiber handler function for processing deposit requests.
func Deposit(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authSvc := service.NewAuthService(uowFactory, strategy)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
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
		accountSvc := service.NewAccountService(uowFactory)

		tx, err := accountSvc.Deposit(userID, id, request.Amount)
		if err != nil {
			log.Errorf("Failed to deposit: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to deposit", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Deposit successful", Data: tx})
	}
}

// Withdraw returns a Fiber handler for processing account withdrawal requests.
// It expects a UnitOfWork factory function as a dependency for transactional operations.
//
// The handler performs the following steps:
//  1. Retrieves the current user ID from the request context.
//  2. Parses the account ID from the route parameters.
//  3. Parses the withdrawal amount from the request body.
//  4. Calls the AccountService.Withdraw method to process the withdrawal.
//  5. Returns the transaction details as a JSON response on success.
//
// Error responses are returned in JSON format with appropriate status codes
// if any step fails (e.g., invalid user ID, invalid account ID, parsing errors, or withdrawal errors).
//
// Parameters:
//   - uowFactory: A function that returns a new UnitOfWork and error.
//
// Returns:
//   - fiber.Handler: The HTTP handler for the withdrawal endpoint.
func Withdraw(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authSvc := service.NewAuthService(uowFactory, strategy)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
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
		accountSvc := service.NewAccountService(uowFactory)
		tx, err := accountSvc.Withdraw(userID, id, request.Amount)
		if err != nil {
			log.Errorf("Failed to withdraw: %v", err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to withdraw", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Withdrawal successful", Data: tx})
	}
}

// GetTransactions returns a Fiber handler that retrieves the list of transactions for a specific account.
// It expects a UnitOfWork factory function as a dependency for service instantiation.
// The handler extracts the current user ID from the request context and parses the account ID from the URL parameters.
// On success, it returns the transactions as a JSON response. On error, it logs the error and returns an appropriate JSON error response.
//
// Route parameters:
//   - id: UUID of the account whose transactions are to be retrieved.
//
// Responses:
//   - 200: JSON array of transactions
//   - 400: Invalid account ID or user ID
//   - 401/403: Unauthorized or forbidden
//   - 500: Internal server error
//
// Example usage:
//
//	app.Get("/accounts/:id/transactions", GetTransactions(uowFactory))
func GetTransactions(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authSvc := service.NewAuthService(uowFactory, strategy)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
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

		service := service.NewAccountService(uowFactory)
		tx, err := service.GetTransactions(userID, id)
		if err != nil {
			log.Errorf("Failed to list transactions for account ID %s: %v", id, err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to list transactions", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Transactions fetched", Data: tx})
	}
}

func GetBalance(uowFactory func() (repository.UnitOfWork, error), strategy service.AuthStrategy) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authSvc := service.NewAuthService(uowFactory, strategy)
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ErrorResponseJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
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
		service := service.NewAccountService(uowFactory)

		balance, err := service.GetBalance(userID, id)
		if err != nil {
			log.Errorf("Failed to fetch balance for account ID %s: %v", id, err)
			status := ErrorToStatusCode(err)
			return ErrorResponseJSON(c, status, "Failed to fetch balance", err.Error())
		}
		return c.Status(fiber.StatusOK).JSON(Response{Status: fiber.StatusOK, Message: "Balance fetched", Data: fiber.Map{"balance": balance}})
	}
}
