// AccountRoutes registers HTTP routes for account-related operations using the Fiber web framework.
// It sets up endpoints for creating accounts, depositing and withdrawing funds, retrieving account balances,
// and listing account transactions. All routes are protected by authentication middleware and require a valid user context.
//
//	@param app The Fiber application instance to register routes on.
//	@param accountSvc A pointer to the AccountService.
//	@param authSvc A pointer to the AuthService.
//
// Routes:
//   - POST   /account                   : Create a new account for the authenticated user.
//   - POST   /account/:id/deposit       : Deposit funds into the specified account.
//   - POST   /account/:id/withdraw      : Withdraw funds from the specified account.
//   - GET    /account/:id/balance       : Retrieve the balance of the specified account.
//   - GET    /account/:id/transactions  : List transactions for the specified account.

package account

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/middleware"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AccountRoutes(app *fiber.App, accountSvc *accountsvc.AccountService, authSvc *authsvc.AuthService, cfg *config.AppConfig) {
	app.Post("/account", middleware.JwtProtected(cfg.Jwt), CreateAccount(accountSvc, authSvc))
	app.Post("/account/:id/deposit", middleware.JwtProtected(cfg.Jwt), Deposit(accountSvc, authSvc))
	app.Post("/account/:id/withdraw", middleware.JwtProtected(cfg.Jwt), Withdraw(accountSvc, authSvc))
	app.Get("/account/:id/balance", middleware.JwtProtected(cfg.Jwt), GetBalance(accountSvc, authSvc))
	app.Get("/account/:id/transactions", middleware.JwtProtected(cfg.Jwt), GetTransactions(accountSvc, authSvc))
}

// CreateAccount returns a Fiber handler for creating a new account for the current user.
// It extracts the user ID from the request context, initializes the account service using the provided
// UnitOfWork factory, and attempts to create a new account. On success, it returns the created account as JSON.
// On failure, it logs the error and returns an appropriate error response.
// @Summary Create a new account
// @Description Create a new account for the authenticated user
// @Tags accounts
// @Accept json
// @Produce json
// @Success 201 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /account [post]
// @Security Bearer
func CreateAccount(
	accountSvc *accountsvc.AccountService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Infof("Creating new account")
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}
		input, err := common.BindAndValidate[CreateAccountRequest](c)
		if input == nil {
			return err // error response already written
		}
		currencyCode := currency.Code("USD")
		if input != nil && input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		a, err := accountSvc.CreateAccountWithCurrency(userID, currencyCode)
		if err != nil {
			log.Errorf("Failed to create account: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to create account", err)
		}
		log.Infof("Account created: %+v", a)
		return common.SuccessResponseJSON(c, fiber.StatusCreated, "Account created", a)
	}
}

// Deposit returns a Fiber handler for depositing an amount into a user's account.
// It expects a UnitOfWork factory function as a dependency for transactional operations.
// The handler parses the current user ID from the request context, validates the account ID from the URL,
// and parses the deposit amount from the request body. If successful, it performs the deposit operation
// using the AccountService and returns the transaction as JSON. On error, it logs the issue and returns
// an appropriate JSON error response.
// @Summary Deposit funds into an account
// @Description Deposit a specified amount into the user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body DepositRequest true "Deposit request with amount"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /account/{id}/deposit [post]
// @Security Bearer
func Deposit(
	accountSvc *accountsvc.AccountService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}
		accountID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid account ID", err, "Account ID must be a valid UUID", fiber.StatusBadRequest)
		}
		input, err := common.BindAndValidate[DepositRequest](c)
		if input == nil {
			return err // error response already written
		}
		currencyCode := currency.Code("USD")
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		tx, convInfo, err := accountSvc.Deposit(userID, accountID, input.Amount, currencyCode)
		if err != nil {
			log.Errorf("Failed to deposit: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to deposit", err)
		}
		if convInfo != nil {
			resp := ToConversionResponseDTO(tx, convInfo)
			return common.SuccessResponseJSON(c, fiber.StatusOK, "Deposit successful (converted)", resp)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Deposit successful", ToTransactionDTO(tx))
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
// @Summary Withdraw funds from an account
// @Description Withdraw a specified amount from the user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body WithdrawRequest true "Withdrawal request with amount"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /account/{id}/withdraw [post]
// @Security Bearer
func Withdraw(
	accountSvc *accountsvc.AccountService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}
		accountID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for withdrawal: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid account ID", err, "Account ID must be a valid UUID", fiber.StatusBadRequest)
		}
		input, err := common.BindAndValidate[WithdrawRequest](c)
		if input == nil {
			return err // error response already written
		}
		currencyCode := currency.Code("USD")
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		tx, convInfo, err := accountSvc.Withdraw(userID, accountID, input.Amount, currencyCode)
		if err != nil {
			log.Errorf("Failed to withdraw: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to withdraw", err)
		}
		if convInfo != nil {
			resp := ToConversionResponseDTO(tx, convInfo)
			return common.SuccessResponseJSON(c, fiber.StatusOK, "Withdrawal successful (converted)", resp)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Withdrawal successful", ToTransactionDTO(tx))
	}
}

// GetTransactions returns a Fiber handler that retrieves the list of transactions for a specific account.
// It expects a UnitOfWork factory function as a dependency for service instantiation.
// The handler extracts the current user ID from the request context and parses the account ID from the URL parameters.
// On success, it returns the transactions as a JSON response. On error, it logs the error and returns an appropriate JSON error response.
// @Summary Get account transactions
// @Description Retrieve the list of transactions for a specific account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /account/{id}/transactions [get]
// @Security Bearer
func GetTransactions(
	accountSvc *accountsvc.AccountService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for transactions: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid account ID", err, "Account ID must be a valid UUID", fiber.StatusBadRequest)
		}

		tx, err := accountSvc.GetTransactions(userID, id)
		if err != nil {
			log.Errorf("Failed to list transactions for account ID %s: %v", id, err)
			return common.ProblemDetailsJSON(c, "Failed to list transactions", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Transactions fetched", tx)
	}
}

// GetBalance returns a Fiber handler for retrieving the balance of a specific account.
// It expects a UnitOfWork factory function as a dependency for service instantiation.
// The handler extracts the current user ID from the request context and parses the account ID from the URL parameters.
// On success, it returns the account balance as a JSON response. On error, it logs the error and returns an appropriate JSON error response.
// @Summary Get account balance
// @Description Retrieve the balance of a specific account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /account/{id}/balance [get]
// @Security Bearer
func GetBalance(
	accountSvc *accountsvc.AccountService,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for balance: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid account ID", err, "Account ID must be a valid UUID", fiber.StatusBadRequest)
		}

		balance, err := accountSvc.GetBalance(userID, id)
		if err != nil {
			log.Errorf("Failed to fetch balance for account ID %s: %v", id, err)
			return common.ProblemDetailsJSON(c, "Failed to fetch balance", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Balance fetched", fiber.Map{"balance": balance})
	}
}
