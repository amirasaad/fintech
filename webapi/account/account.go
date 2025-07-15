package account

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/handler"
	"github.com/amirasaad/fintech/pkg/middleware"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Routes registers HTTP routes for account-related operations using the Fiber web framework.
// It sets up endpoints for creating accounts, depositing and withdrawing funds, retrieving account balances,
// and listing account transactions. All routes are protected by authentication middleware and require a valid user context.
//
// Routes:
//   - POST   /account                   : Create a new account for the authenticated user.
//   - POST   /account/:id/deposit       : Deposit funds into the specified account.
//   - POST   /account/:id/withdraw      : Withdraw funds from the specified account.
//   - GET    /account/:id/balance       : Retrieve the balance of the specified account.
//   - GET    /account/:id/transactions  : List transactions for the specified account.
func Routes(app *fiber.App, accountSvc *accountsvc.Service, authSvc *authsvc.AuthService, cfg *config.AppConfig) {
	app.Post("/account", middleware.JwtProtected(cfg.Jwt), CreateAccount(accountSvc, authSvc))
	app.Post("/account/:id/deposit", middleware.JwtProtected(cfg.Jwt), Deposit(accountSvc, authSvc))
	app.Post("/account/:id/withdraw", middleware.JwtProtected(cfg.Jwt), Withdraw(accountSvc, authSvc))
	app.Post("/account/:id/transfer", middleware.JwtProtected(cfg.Jwt), Transfer(accountSvc, authSvc))
	app.Get("/account/:id/balance", middleware.JwtProtected(cfg.Jwt), GetBalance(accountSvc, authSvc))
	app.Get("/account/:id/transactions", middleware.JwtProtected(cfg.Jwt), GetTransactions(accountSvc, authSvc))
}

// CreateAccount returns a Fiber handler for creating a new account for the current user.
// It extracts the user ID from the request context, initializes the account service using the provided
// UnitOfWork factory, and attempts to create a new account. On success, it returns the created account as JSON.
// On failure, it logs the error and returns an appropriate error response.
// @Summary Create a new account
// @Description Creates a new account for the authenticated user. You can specify the currency for the account. Returns the created account details.
// @Tags accounts
// @Accept json
// @Produce json
// @Success 201 {object} common.Response "Account created successfully"
// @Failure 400 {object} common.ProblemDetails "Invalid request"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 429 {object} common.ProblemDetails "Too many requests"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /account [post]
// @Security Bearer
func CreateAccount(
	accountSvc *accountsvc.Service,
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
		currencyCode := currency.USD
		if input.Currency != "" {
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
// @Description Adds funds to the specified account. Specify the amount, currency, and optional money source. Returns the transaction details.
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body DepositRequest true "Deposit details"
// @Success 200 {object} common.Response "Deposit successful"
// @Failure 400 {object} common.ProblemDetails "Invalid request"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 429 {object} common.ProblemDetails "Too many requests"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /account/{id}/deposit [post]
// @Security Bearer
func Deposit(
	accountSvc *accountsvc.Service,
	authSvc *authsvc.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Infof("Deposit handler: called for account %s", c.Params("id"))
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
		currencyCode := currency.USD
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		log.Infof("Deposit handler: calling service for user %s, account %s, amount %v, currency %s, money_source %s", userID, accountID, input.Amount, currencyCode, input.MoneySource)
		err = accountSvc.Deposit(userID, accountID, input.Amount, currencyCode, input.MoneySource)
		if err != nil {
			log.Errorf("Failed to deposit: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to deposit", err)
		}
		resp := fiber.Map{}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Deposit successful", resp)
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
// @Description Withdraws a specified amount from the user's account. Specify the amount and currency. Returns the transaction details.
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body WithdrawRequest true "Withdrawal details"
// @Success 200 {object} common.Response "Withdrawal successful"
// @Failure 400 {object} common.ProblemDetails "Invalid request"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 429 {object} common.ProblemDetails "Too many requests"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /account/{id}/withdraw [post]
// @Security Bearer
func Withdraw(
	accountSvc *accountsvc.Service,
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
		// Validate that at least one field in ExternalTarget is present
		if input.ExternalTarget.BankAccountNumber == "" && input.ExternalTarget.RoutingNumber == "" && input.ExternalTarget.ExternalWalletAddress == "" {
			return common.ProblemDetailsJSON(c, "Invalid external target", nil, "At least one external target field must be provided", fiber.StatusBadRequest)
		}
		currencyCode := currency.Code("USD")
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		handlerTarget := handler.ExternalTarget{
			BankAccountNumber:     input.ExternalTarget.BankAccountNumber,
			RoutingNumber:         input.ExternalTarget.RoutingNumber,
			ExternalWalletAddress: input.ExternalTarget.ExternalWalletAddress,
		}
		err = accountSvc.Withdraw(userID, accountID, input.Amount, currencyCode, handlerTarget)
		if err != nil {
			log.Errorf("Failed to withdraw: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to withdraw", err)
		}
		resp := fiber.Map{}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Withdrawal successful", resp)
	}
}

// Transfer returns a Fiber handler for transferring funds between accounts.
// @Summary Transfer funds between accounts
// @Description Transfers a specified amount from one account to another. Specify the source and destination account IDs, amount, and currency. Returns the transaction details.
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Source Account ID"
// @Param request body TransferRequest true "Transfer details"
// @Success 200 {object} common.Response "Transfer successful"
// @Failure 400 {object} common.ProblemDetails "Invalid request"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 422 {object} common.ProblemDetails "Unprocessable entity"
// @Failure 429 {object} common.ProblemDetails "Too many requests"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /account/{id}/transfer [post]
// @Security Bearer
func Transfer(accountSvc *accountsvc.Service, authSvc *authsvc.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Infof("Transfer handler: called for account %s", c.Params("id"))
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return common.ProblemDetailsJSON(c, "Unauthorized", nil, "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid user ID", err)
		}
		sourceAccountID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid source account ID for transfer: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid account ID", err, "Account ID must be a valid UUID", fiber.StatusBadRequest)
		}
		input, err := common.BindAndValidate[TransferRequest](c)
		if input == nil {
			return err // error response already written
		}
		destAccountID, err := uuid.Parse(input.DestinationAccountID)
		if err != nil {
			log.Errorf("Invalid destination account ID for transfer: %v", err)
			return common.ProblemDetailsJSON(c, "Invalid destination account ID", err, "Destination Account ID must be a valid UUID", fiber.StatusBadRequest)
		}
		currencyCode := currency.USD
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		log.Infof("Transfer handler: calling service for user %s, source account %s, dest account %s, amount %v, currency %s", userID, sourceAccountID, destAccountID, input.Amount, currencyCode)
		err = accountSvc.Transfer(userID, sourceAccountID, destAccountID, input.Amount, currencyCode)
		if err != nil {
			log.Errorf("Failed to transfer: %v", err)
			return common.ProblemDetailsJSON(c, "Failed to transfer", err)
		}
		resp := fiber.Map{}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Transfer successful", resp)
	}
}

// GetTransactions returns a Fiber handler that retrieves the list of transactions for a specific account.
// It expects a UnitOfWork factory function as a dependency for service instantiation.
// The handler extracts the current user ID from the request context and parses the account ID from the URL parameters.
// On success, it returns the transactions as a JSON response. On error, it logs the error and returns an appropriate JSON error response.
// @Summary Get account transactions
// @Description Retrieves a list of transactions for the specified account. Returns an array of transaction details.
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} common.Response "Transactions fetched"
// @Failure 400 {object} common.ProblemDetails "Invalid request"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 429 {object} common.ProblemDetails "Too many requests"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /account/{id}/transactions [get]
// @Security Bearer
func GetTransactions(
	accountSvc *accountsvc.Service,
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
		dtos := make([]*TransactionDTO, 0, len(tx))
		for _, t := range tx {
			dtos = append(dtos, ToTransactionDTO(t))
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Transactions fetched", dtos)
	}
}

// GetBalance returns a Fiber handler for retrieving the balance of a specific account.
// It expects a UnitOfWork factory function as a dependency for service instantiation.
// The handler extracts the current user ID from the request context and parses the account ID from the URL parameters.
// On success, it returns the account balance as a JSON response. On error, it logs the error and returns an appropriate JSON error response.
// @Summary Get account balance
// @Description Retrieves the current balance for the specified account. Returns the balance amount and currency.
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} common.Response "Balance fetched"
// @Failure 400 {object} common.ProblemDetails "Invalid request"
// @Failure 401 {object} common.ProblemDetails "Unauthorized"
// @Failure 429 {object} common.ProblemDetails "Too many requests"
// @Failure 500 {object} common.ProblemDetails "Internal server error"
// @Router /account/{id}/balance [get]
// @Security Bearer
func GetBalance(
	accountSvc *accountsvc.Service,
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
