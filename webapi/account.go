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

package webapi

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/middleware"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type CreateAccountRequest struct {
	Currency string `json:"currency" validate:"omitempty,len=3,uppercase"`
}

type DepositRequest struct {
	Amount   float64 `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"omitempty,len=3,uppercase"`
}

type WithdrawRequest struct {
	Amount   float64 `json:"amount" xml:"amount" form:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"omitempty,len=3,uppercase"`
}

// ConversionResponse wraps a transaction and conversion details if a currency conversion occurred.
type ConversionResponse struct {
	Transaction       *domain.Transaction `json:"transaction"`
	OriginalAmount    float64             `json:"original_amount,omitempty"`
	OriginalCurrency  string              `json:"original_currency,omitempty"`
	ConvertedAmount   float64             `json:"converted_amount,omitempty"`
	ConvertedCurrency string              `json:"converted_currency,omitempty"`
	ConversionRate    float64             `json:"conversion_rate,omitempty"`
}

// TransactionDTO is the API response representation of a transaction.
type TransactionDTO struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Balance   float64 `json:"balance"`
	CreatedAt string  `json:"created_at"`
	Currency  string  `json:"currency"`

	// Conversion fields (only present if conversion occurred)
	OriginalAmount   *float64 `json:"original_amount,omitempty"`
	OriginalCurrency *string  `json:"original_currency,omitempty"`
	ConversionRate   *float64 `json:"conversion_rate,omitempty"`
}

// ConversionResponseDTO wraps a transaction and conversion details for API responses.
type ConversionResponseDTO struct {
	Transaction       *TransactionDTO `json:"transaction"`
	OriginalAmount    float64         `json:"original_amount,omitempty"`
	OriginalCurrency  string          `json:"original_currency,omitempty"`
	ConvertedAmount   float64         `json:"converted_amount,omitempty"`
	ConvertedCurrency string          `json:"converted_currency,omitempty"`
	ConversionRate    float64         `json:"conversion_rate,omitempty"`
}

// ToTransactionDTO maps a domain.Transaction to a TransactionDTO.
func ToTransactionDTO(tx *domain.Transaction) *TransactionDTO {
	if tx == nil {
		return nil
	}
	dto := &TransactionDTO{
		ID:        tx.ID.String(),
		UserID:    tx.UserID.String(),
		AccountID: tx.AccountID.String(),
		Amount:    float64(tx.Amount) / 100.0, // assuming cents
		Balance:   float64(tx.Balance) / 100.0,
		CreatedAt: tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Currency:  string(tx.Currency),
	}

	// Include conversion fields if they exist
	if tx.OriginalAmount != nil {
		dto.OriginalAmount = tx.OriginalAmount
	}
	if tx.OriginalCurrency != nil {
		dto.OriginalCurrency = tx.OriginalCurrency
	}
	if tx.ConversionRate != nil {
		dto.ConversionRate = tx.ConversionRate
	}

	return dto
}

// ToConversionResponseDTO maps a transaction and conversion info to a ConversionResponseDTO.
func ToConversionResponseDTO(tx *domain.Transaction, convInfo *domain.ConversionInfo) *ConversionResponseDTO {
	// If conversion info is provided (from service layer), use it
	if convInfo != nil {
		return &ConversionResponseDTO{
			Transaction:       ToTransactionDTO(tx),
			OriginalAmount:    convInfo.OriginalAmount,
			OriginalCurrency:  convInfo.OriginalCurrency,
			ConvertedAmount:   convInfo.ConvertedAmount,
			ConvertedCurrency: convInfo.ConvertedCurrency,
			ConversionRate:    convInfo.ConversionRate,
		}
	}

	// If no conversion info provided but transaction has stored conversion data, use that
	if tx.OriginalAmount != nil && tx.OriginalCurrency != nil && tx.ConversionRate != nil {
		return &ConversionResponseDTO{
			Transaction:       ToTransactionDTO(tx),
			OriginalAmount:    *tx.OriginalAmount,
			OriginalCurrency:  *tx.OriginalCurrency,
			ConvertedAmount:   float64(tx.Amount) / 100.0, // Convert from cents
			ConvertedCurrency: string(tx.Currency),
			ConversionRate:    *tx.ConversionRate,
		}
	}

	// No conversion occurred
	return nil
}

func AccountRoutes(app *fiber.App, accountSvc *service.AccountService, authSvc *service.AuthService, cfg *config.AppConfig) {
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
// @Success 201 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /account [post]
// @Security Bearer
func CreateAccount(
	accountSvc *service.AccountService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Infof("Creating new account")
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		input, _ := BindAndValidate[CreateAccountRequest](c) // ignore error, currency is optional
		currencyCode := currency.Code("USD")
		if input != nil && input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		a, err := accountSvc.CreateAccountWithCurrency(userID, currencyCode)
		if err != nil {
			log.Errorf("Failed to create account: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "Failed to create account", err.Error())
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
// @Summary Deposit funds into an account
// @Description Deposit a specified amount into the user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param request body DepositRequest true "Deposit request with amount"
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /account/{id}/deposit [post]
// @Security Bearer
func Deposit(
	accountSvc *service.AccountService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		accountID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for deposit: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}
		input, err := BindAndValidate[DepositRequest](c)
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "invalid input", err.Error())
		}
		currencyCode := currency.Code("USD")
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		tx, convInfo, err := accountSvc.Deposit(userID, accountID, input.Amount, currencyCode)
		if err != nil {
			log.Errorf("Failed to deposit: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "Failed to deposit", err.Error())
		}
		if convInfo != nil {
			resp := ToConversionResponseDTO(tx, convInfo)
			return c.JSON(Response{Status: fiber.StatusOK, Message: "Deposit successful (converted)", Data: resp})
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Deposit successful", Data: ToTransactionDTO(tx)})
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
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /account/{id}/withdraw [post]
// @Security Bearer
func Withdraw(
	accountSvc *service.AccountService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		accountID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for withdrawal: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}
		input, err := BindAndValidate[WithdrawRequest](c)
		if err != nil {
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "invalid input", err.Error())
		}
		currencyCode := currency.Code("USD")
		if input.Currency != "" {
			currencyCode = currency.Code(input.Currency)
		}
		tx, convInfo, err := accountSvc.Withdraw(userID, accountID, input.Amount, currencyCode)
		if err != nil {
			log.Errorf("Failed to withdraw: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "Failed to withdraw", err.Error())
		}
		if convInfo != nil {
			resp := ToConversionResponseDTO(tx, convInfo)
			return c.JSON(Response{Status: fiber.StatusOK, Message: "Withdrawal successful (converted)", Data: resp})
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Withdrawal successful", Data: ToTransactionDTO(tx)})
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
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /account/{id}/transactions [get]
// @Security Bearer
func GetTransactions(
	accountSvc *service.AccountService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for transactions: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}

		tx, err := accountSvc.GetTransactions(userID, id)
		if err != nil {
			log.Errorf("Failed to list transactions for account ID %s: %v", id, err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "Failed to list transactions", err.Error())
		}
		return c.JSON(Response{Status: fiber.StatusOK, Message: "Transactions fetched", Data: tx})
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
// @Success 200 {object} Response
// @Failure 400 {object} ProblemDetails
// @Failure 401 {object} ProblemDetails
// @Failure 429 {object} ProblemDetails
// @Failure 500 {object} ProblemDetails
// @Router /account/{id}/balance [get]
// @Security Bearer
func GetBalance(
	accountSvc *service.AccountService,
	authSvc *service.AuthService,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, ok := c.Locals("user").(*jwt.Token)
		if !ok {
			return ProblemDetailsJSON(c, fiber.StatusUnauthorized, "unauthorized", "missing user context")
		}
		userID, err := authSvc.GetCurrentUserId(token)
		if err != nil {
			log.Errorf("Failed to parse user ID from token: %v", err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "invalid user ID", err.Error())
		}
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			log.Errorf("Invalid account ID for balance: %v", err)
			return ProblemDetailsJSON(c, fiber.StatusBadRequest, "Invalid account ID", err.Error())
		}

		balance, err := accountSvc.GetBalance(userID, id)
		if err != nil {
			log.Errorf("Failed to fetch balance for account ID %s: %v", id, err)
			status := ErrorToStatusCode(err)
			return ProblemDetailsJSON(c, status, "Failed to fetch balance", err.Error())
		}
		return c.Status(fiber.StatusOK).JSON(Response{Status: fiber.StatusOK, Message: "Balance fetched", Data: fiber.Map{"balance": balance}})
	}
}
