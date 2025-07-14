package currency

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/middleware"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	currencysvc "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/amirasaad/fintech/webapi/common"
	"github.com/gofiber/fiber/v2"
)

// Routes registers HTTP routes for currency-related operations.
// Sets up endpoints for listing, searching, and managing currencies.
func Routes(app *fiber.App, currencySvc *currencysvc.CurrencyService, authSvc *authsvc.AuthService, cfg *config.AppConfig) {
	currencyGroup := app.Group("/api/currencies")

	// Public endpoints
	currencyGroup.Get("/", ListCurrencies(currencySvc))
	currencyGroup.Get("/supported", ListSupportedCurrencies(currencySvc))
	currencyGroup.Get("/:code", GetCurrency(currencySvc))
	currencyGroup.Get("/:code/supported", CheckCurrencySupported(currencySvc))
	currencyGroup.Get("/search", SearchCurrencies(currencySvc))
	currencyGroup.Get("/region/:region", SearchCurrenciesByRegion(currencySvc))
	currencyGroup.Get("/statistics", GetCurrencyStatistics(currencySvc))
	currencyGroup.Get("/default", GetDefaultCurrency(currencySvc))

	// Admin endpoints (require authentication)
	adminGroup := currencyGroup.Group("/admin")
	adminGroup.Post("/", middleware.JwtProtected(cfg.Jwt), RegisterCurrency(currencySvc))
	adminGroup.Delete("/:code", middleware.JwtProtected(cfg.Jwt), UnregisterCurrency(currencySvc))
	adminGroup.Put("/:code/activate", middleware.JwtProtected(cfg.Jwt), ActivateCurrency(currencySvc))
	adminGroup.Put("/:code/deactivate", middleware.JwtProtected(cfg.Jwt), DeactivateCurrency(currencySvc))
}

// ListCurrencies returns a Fiber handler for listing all available currencies.
// @Summary List all currencies
// @Description Get a list of all available currencies in the system
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 429 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /currency [get]
// @Security Bearer
func ListCurrencies(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		currencies, err := currencySvc.ListAllCurrencies(c.Context())
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to list currencies", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currencies fetched successfully", currencies)
	}
}

// ListSupportedCurrencies returns all supported currency codes
// @Summary List supported currencies
// @Description Get all supported currency codes
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/supported [get]
func ListSupportedCurrencies(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		currencies, err := currencySvc.ListSupportedCurrencies(c.Context())
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to list supported currencies", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Supported currencies fetched successfully", currencies)
	}
}

// GetCurrency returns currency information by code
// @Summary Get currency by code
// @Description Get currency information by ISO 4217 code
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code (e.g., USD, EUR)"
// @Success 200 {object} currency.CurrencyMeta
// @Failure 400 {object} common.ProblemDetails
// @Failure 404 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/{code} [get]
func GetCurrency(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return common.ProblemDetailsJSON(c, "Currency code is required", nil, "Missing currency code", fiber.StatusBadRequest)
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return common.ProblemDetailsJSON(c, "Invalid currency code", err, "Currency code must be a valid ISO 4217 code", fiber.StatusBadRequest)
		}

		currency, err := currencySvc.GetCurrency(c.Context(), code)
		if err != nil {
			return common.ProblemDetailsJSON(c, "Currency not found", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currency fetched successfully", currency)
	}
}

// CheckCurrencySupported checks if a currency is supported
// @Summary Check if currency is supported
// @Description Check if a currency code is supported
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code (e.g., USD, EUR)"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Router /api/currencies/{code}/supported [get]
func CheckCurrencySupported(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return common.ProblemDetailsJSON(c, "Currency code is required", nil)
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return common.ProblemDetailsJSON(c, "Invalid currency code", err)
		}

		supported := currencySvc.IsCurrencySupported(c.Context(), code)
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currency support checked successfully", fiber.Map{"code": code, "supported": supported})
	}
}

// SearchCurrencies searches for currencies by name
// @Summary Search currencies
// @Description Search for currencies by name
// @Tags currencies
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {array} currency.CurrencyMeta
// @Failure 400 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/search [get]
func SearchCurrencies(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return common.ProblemDetailsJSON(c, "Search query is required", nil, "Missing search query", fiber.StatusBadRequest)
		}

		currencies, err := currencySvc.SearchCurrencies(c.Context(), query)
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to search currencies", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currencies searched successfully", currencies)
	}
}

// SearchCurrenciesByRegion searches for currencies by region
// @Summary Search currencies by region
// @Description Search for currencies by region
// @Tags currencies
// @Accept json
// @Produce json
// @Param region path string true "Region name"
// @Success 200 {array} currency.CurrencyMeta
// @Failure 400 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/region/{region} [get]
func SearchCurrenciesByRegion(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		region := c.Params("region")
		if region == "" {
			return common.ProblemDetailsJSON(c, "Region is required", nil, "Missing region", fiber.StatusBadRequest)
		}

		currencies, err := currencySvc.SearchCurrenciesByRegion(c.Context(), region)
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to search currencies by region", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currencies by region fetched successfully", currencies)
	}
}

// GetCurrencyStatistics returns currency statistics
// @Summary Get currency statistics
// @Description Get currency registry statistics
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {object} common.Response
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/statistics [get]
func GetCurrencyStatistics(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := currencySvc.GetCurrencyStatistics(c.Context())
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to get currency statistics", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currency statistics fetched successfully", stats)
	}
}

// GetDefaultCurrency returns the default currency information
// @Summary Get default currency
// @Description Get the default currency information
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {object} currency.CurrencyMeta
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/default [get]
func GetDefaultCurrency(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defaultCurrency, err := currencySvc.GetDefaultCurrency(c.Context())
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to get default currency", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Default currency fetched successfully", defaultCurrency)
	}
}

// RegisterCurrency registers a new currency (admin only)
// @Summary Register currency
// @Description Register a new currency (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param currency body RegisterCurrencyRequest true "Currency information"
// @Success 201 {object} currency.CurrencyMeta
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 409 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/admin [post]
func RegisterCurrency(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := common.BindAndValidate[RegisterCurrencyRequest](c)
		if err != nil {
			return nil // Error already written by BindAndValidate
		}

		// Validate currency code format
		if err = currencySvc.ValidateCurrencyCode(c.Context(), input.Code); err != nil {
			return common.ProblemDetailsJSON(c, "Invalid currency code", err)
		}

		currencyMeta := currency.CurrencyMeta{
			Code:     input.Code,
			Name:     input.Name,
			Symbol:   input.Symbol,
			Decimals: input.Decimals,
			Country:  input.Country,
			Region:   input.Region,
			Active:   input.Active,
			Metadata: input.Metadata,
		}

		if err = currencySvc.RegisterCurrency(c.Context(), currencyMeta); err != nil {
			if err == currency.ErrCurrencyExists {
				return common.ProblemDetailsJSON(c, "Failed to register currency", err)
			}
			return common.ProblemDetailsJSON(c, "Failed to register currency", err)
		}

		// Get the registered currency
		registered, err := currencySvc.GetCurrency(c.Context(), input.Code)
		if err != nil {
			return common.ProblemDetailsJSON(c, "Failed to retrieve registered currency", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusCreated, "Currency registered successfully", registered)
	}
}

// UnregisterCurrency removes a currency from the registry (admin only)
// @Summary Unregister currency
// @Description Remove a currency from the registry (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 404 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/admin/{code} [delete]
func UnregisterCurrency(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return common.ProblemDetailsJSON(c, "Currency code is required", nil)
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return common.ProblemDetailsJSON(c, "Invalid currency code", err)
		}

		if err := currencySvc.UnregisterCurrency(c.Context(), code); err != nil {
			if err == currency.ErrCurrencyNotFound {
				return common.ProblemDetailsJSON(c, "Failed to unregister currency", err)
			}
			return common.ProblemDetailsJSON(c, "Failed to unregister currency", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currency unregistered successfully", fiber.Map{"code": code})
	}
}

// ActivateCurrency activates a currency (admin only)
// @Summary Activate currency
// @Description Activate a currency (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 404 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/admin/{code}/activate [put]
func ActivateCurrency(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return common.ProblemDetailsJSON(c, "Currency code is required", nil)
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return common.ProblemDetailsJSON(c, "Invalid currency code", err)
		}

		if err := currencySvc.ActivateCurrency(c.Context(), code); err != nil {
			if err == currency.ErrCurrencyNotFound {
				return common.ProblemDetailsJSON(c, "Failed to activate currency", err)
			}
			return common.ProblemDetailsJSON(c, "Failed to activate currency", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currency activated successfully", fiber.Map{"code": code})
	}
}

// DeactivateCurrency deactivates a currency (admin only)
// @Summary Deactivate currency
// @Description Deactivate a currency (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.ProblemDetails
// @Failure 401 {object} common.ProblemDetails
// @Failure 404 {object} common.ProblemDetails
// @Failure 500 {object} common.ProblemDetails
// @Router /api/currencies/admin/{code}/deactivate [put]
func DeactivateCurrency(currencySvc *currencysvc.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return common.ProblemDetailsJSON(c, "Currency code is required", nil)
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return common.ProblemDetailsJSON(c, "Invalid currency code", err)
		}

		if err := currencySvc.DeactivateCurrency(c.Context(), code); err != nil {
			if err == currency.ErrCurrencyNotFound {
				return common.ProblemDetailsJSON(c, "Failed to deactivate currency", err)
			}
			return common.ProblemDetailsJSON(c, "Failed to deactivate currency", err)
		}
		return common.SuccessResponseJSON(c, fiber.StatusOK, "Currency deactivated successfully", fiber.Map{"code": code})
	}
}
