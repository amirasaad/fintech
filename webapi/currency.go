package webapi

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/middleware"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/gofiber/fiber/v2"
)

// CurrencyRoutes sets up currency-related routes
func CurrencyRoutes(app *fiber.App, currencySvc *service.CurrencyService, authSvc *service.AuthService, cfg *config.AppConfig) {
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

// ListCurrencies returns all registered currencies
// @Summary List all currencies
// @Description Get all registered currencies with full metadata
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {array} currency.CurrencyMeta
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies [get]
func ListCurrencies(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		currencies, err := currencySvc.ListAllCurrencies(c.Context())
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to list currencies", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currencies fetched successfully",
			Data:    currencies,
		})
	}
}

// ListSupportedCurrencies returns all supported currency codes
// @Summary List supported currencies
// @Description Get all supported currency codes
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/supported [get]
func ListSupportedCurrencies(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		currencies, err := currencySvc.ListSupportedCurrencies(c.Context())
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to list supported currencies", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Supported currencies fetched successfully",
			Data:    currencies,
		})
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
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/{code} [get]
func GetCurrency(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Currency code is required", "Missing currency code")
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid currency code", err.Error())
		}

		currency, err := currencySvc.GetCurrency(c.Context(), code)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusNotFound, "Currency not found", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currency fetched successfully",
			Data:    currency,
		})
	}
}

// CheckCurrencySupported checks if a currency is supported
// @Summary Check if currency is supported
// @Description Check if a currency code is supported
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code (e.g., USD, EUR)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /api/currencies/{code}/supported [get]
func CheckCurrencySupported(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Currency code is required", "Missing currency code")
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid currency code", err.Error())
		}

		supported := currencySvc.IsCurrencySupported(c.Context(), code)

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currency support checked successfully",
			Data: fiber.Map{
				"code":      code,
				"supported": supported,
			},
		})
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
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/search [get]
func SearchCurrencies(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Search query is required", "Missing search query")
		}

		currencies, err := currencySvc.SearchCurrencies(c.Context(), query)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to search currencies", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currencies searched successfully",
			Data:    currencies,
		})
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
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/region/{region} [get]
func SearchCurrenciesByRegion(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		region := c.Params("region")
		if region == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Region is required", "Missing region")
		}

		currencies, err := currencySvc.SearchCurrenciesByRegion(c.Context(), region)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to search currencies by region", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currencies by region fetched successfully",
			Data:    currencies,
		})
	}
}

// GetCurrencyStatistics returns currency statistics
// @Summary Get currency statistics
// @Description Get currency registry statistics
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/statistics [get]
func GetCurrencyStatistics(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := currencySvc.GetCurrencyStatistics(c.Context())
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to get currency statistics", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currency statistics fetched successfully",
			Data:    stats,
		})
	}
}

// GetDefaultCurrency returns the default currency information
// @Summary Get default currency
// @Description Get the default currency information
// @Tags currencies
// @Accept json
// @Produce json
// @Success 200 {object} currency.CurrencyMeta
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/default [get]
func GetDefaultCurrency(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defaultCurrency, err := currencySvc.GetDefaultCurrency(c.Context())
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to get default currency", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Default currency fetched successfully",
			Data:    defaultCurrency,
		})
	}
}

// RegisterCurrencyRequest represents the request body for registering a currency
type RegisterCurrencyRequest struct {
	Code     string            `json:"code" validate:"required"`
	Name     string            `json:"name" validate:"required"`
	Symbol   string            `json:"symbol" validate:"required"`
	Decimals int               `json:"decimals" validate:"required,min=0,max=8"`
	Country  string            `json:"country,omitempty"`
	Region   string            `json:"region,omitempty"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RegisterCurrency registers a new currency (admin only)
// @Summary Register currency
// @Description Register a new currency (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param currency body RegisterCurrencyRequest true "Currency information"
// @Success 201 {object} currency.CurrencyMeta
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/admin [post]
func RegisterCurrency(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		input, err := BindAndValidate[RegisterCurrencyRequest](c)
		if err != nil {
			return nil // Error already written by helper
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), input.Code); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid currency code", err.Error())
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

		if err := currencySvc.RegisterCurrency(c.Context(), currencyMeta); err != nil {
			status := fiber.StatusInternalServerError
			if err == currency.ErrCurrencyExists {
				status = fiber.StatusConflict
			}
			return ErrorResponseJSON(c, status, "Failed to register currency", err.Error())
		}

		// Get the registered currency
		registered, err := currencySvc.GetCurrency(c.Context(), input.Code)
		if err != nil {
			return ErrorResponseJSON(c, fiber.StatusInternalServerError, "Failed to retrieve registered currency", err.Error())
		}

		return c.Status(fiber.StatusCreated).JSON(Response{
			Status:  fiber.StatusCreated,
			Message: "Currency registered successfully",
			Data:    registered,
		})
	}
}

// UnregisterCurrency removes a currency from the registry (admin only)
// @Summary Unregister currency
// @Description Remove a currency from the registry (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/admin/{code} [delete]
func UnregisterCurrency(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Currency code is required", "Missing currency code")
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid currency code", err.Error())
		}

		if err := currencySvc.UnregisterCurrency(c.Context(), code); err != nil {
			status := fiber.StatusInternalServerError
			if err == currency.ErrCurrencyNotFound {
				status = fiber.StatusNotFound
			}
			return ErrorResponseJSON(c, status, "Failed to unregister currency", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currency unregistered successfully",
			Data: fiber.Map{
				"code": code,
			},
		})
	}
}

// ActivateCurrency activates a currency (admin only)
// @Summary Activate currency
// @Description Activate a currency (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/admin/{code}/activate [put]
func ActivateCurrency(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Currency code is required", "Missing currency code")
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid currency code", err.Error())
		}

		if err := currencySvc.ActivateCurrency(c.Context(), code); err != nil {
			status := fiber.StatusInternalServerError
			if err == currency.ErrCurrencyNotFound {
				status = fiber.StatusNotFound
			}
			return ErrorResponseJSON(c, status, "Failed to activate currency", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currency activated successfully",
			Data: fiber.Map{
				"code": code,
			},
		})
	}
}

// DeactivateCurrency deactivates a currency (admin only)
// @Summary Deactivate currency
// @Description Deactivate a currency (admin only)
// @Tags currencies
// @Accept json
// @Produce json
// @Param code path string true "Currency code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/currencies/admin/{code}/deactivate [put]
func DeactivateCurrency(currencySvc *service.CurrencyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Currency code is required", "Missing currency code")
		}

		// Validate currency code format
		if err := currencySvc.ValidateCurrencyCode(c.Context(), code); err != nil {
			return ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid currency code", err.Error())
		}

		if err := currencySvc.DeactivateCurrency(c.Context(), code); err != nil {
			status := fiber.StatusInternalServerError
			if err == currency.ErrCurrencyNotFound {
				status = fiber.StatusNotFound
			}
			return ErrorResponseJSON(c, status, "Failed to deactivate currency", err.Error())
		}

		return c.JSON(Response{
			Status:  fiber.StatusOK,
			Message: "Currency deactivated successfully",
			Data: fiber.Map{
				"code": code,
			},
		})
	}
}
