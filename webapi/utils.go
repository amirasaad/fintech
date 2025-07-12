package webapi

import (
	"errors"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

// Response defines the standard API response structure for success cases.
type Response struct {
	Status  int    `json:"status"`         // HTTP status code
	Message string `json:"message"`        // Human-readable explanation
	Data    any    `json:"data,omitempty"` // Response data
}

// ProblemDetails follows RFC 9457 Problem Details for HTTP APIs.
type ProblemDetails struct {
	Type     string `json:"type,omitempty"`     // A URI reference that identifies the problem type
	Title    string `json:"title"`              // Short, human-readable summary
	Status   int    `json:"status"`             // HTTP status code
	Detail   string `json:"detail,omitempty"`   // Human-readable explanation
	Instance string `json:"instance,omitempty"` // URI reference that identifies the specific occurrence
	Errors   any    `json:"errors,omitempty"`   // Optional: additional error details
}

// ProblemDetailsJSON writes a problem+json error response with a status code inferred from the error (if present).
// The title is set to the error message (if error), and detail can be a string, error, or structured object.
// Optionally, a status code can be provided as the last argument (int) to override the fallback status.
func ProblemDetailsJSON(c *fiber.Ctx, title string, err error, detailOrStatus ...any) error {
	status := fiber.StatusBadRequest
	pdDetail := ""
	var pdErrors any
	var customStatus *int

	if err != nil {
		status = errorToStatusCode(err)
		title = err.Error()
		pdDetail = err.Error()
	}
	// Check for custom detail or status code in variadic args
	for _, arg := range detailOrStatus {
		switch v := arg.(type) {
		case int:
			customStatus = &v
		case string:
			pdDetail = v
		case error:
			pdDetail = v.Error()
		default:
			pdErrors = v
		}
	}
	// Use custom status if provided
	if customStatus != nil {
		status = *customStatus
	}
	pd := ProblemDetails{
		Status: status,
		Title:  title,
		Detail: pdDetail,
		Errors: pdErrors,
	}
	c.Set(fiber.HeaderContentType, "application/problem+json")
	if err := c.Status(status).JSON(pd); err != nil {
		log.Errorf("ProblemDetailsJSON failed: %v", err)
	}
	return nil
}

// BindAndValidate parses the request body and validates it using go-playground/validator.
// Returns a pointer to the struct (populated), or writes an error response and returns nil.
func BindAndValidate[T any](c *fiber.Ctx) (*T, error) {
	var input T
	if err := c.BodyParser(&input); err != nil {
		return nil, ProblemDetailsJSON(c, "Invalid request body", err, "Request body could not be parsed or has invalid types", fiber.StatusBadRequest) //nolint:errcheck
	}

	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			details := make(map[string]string)
			for _, fe := range ve {
				field := fe.Field()
				msg := fe.Tag()
				details[field] = msg
			}
			return nil, ProblemDetailsJSON(c, "Validation failed", nil, details, fiber.StatusBadRequest) //nolint:errcheck
		}
		ProblemDetailsJSON(c, "Validation failed", err, "Request validation failed", fiber.StatusBadRequest) //nolint:errcheck
		return nil, err
	}
	return &input, nil
}

// SuccessResponseJSON writes a JSON response with the given status, message, and data using the standard Response struct.
// Use for successful API responses (e.g., 200, 201, 202).
func SuccessResponseJSON(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(Response{
		Status:  status,
		Message: message,
		Data:    data,
	})
}

// errorToStatusCode maps domain errors to appropriate HTTP status codes.
func errorToStatusCode(err error) int {
	switch {
	// Account errors
	case errors.Is(err, domain.ErrAccountNotFound):
		return fiber.StatusNotFound
	case errors.Is(err, domain.ErrDepositAmountExceedsMaxSafeInt):
		return fiber.StatusBadRequest
	case errors.Is(err, domain.ErrTransactionAmountMustBePositive):
		return fiber.StatusBadRequest
	case errors.Is(err, domain.ErrWithdrawalAmountMustBePositive):
		return fiber.StatusBadRequest
	case errors.Is(err, domain.ErrInsufficientFunds):
		return fiber.StatusUnprocessableEntity
	// Common errors
	case errors.Is(err, domain.ErrInvalidCurrencyCode):
		return fiber.StatusUnprocessableEntity
	case errors.Is(err, common.ErrInvalidDecimalPlaces):
		return fiber.StatusBadRequest
	case errors.Is(err, common.ErrAmountExceedsMaxSafeInt):
		return fiber.StatusBadRequest
	// Money/currency conversion errors
	case errors.Is(err, domain.ErrExchangeRateUnavailable):
		return fiber.StatusServiceUnavailable
	case errors.Is(err, domain.ErrUnsupportedCurrencyPair):
		return fiber.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrExchangeRateExpired):
		return fiber.StatusServiceUnavailable
	case errors.Is(err, domain.ErrExchangeRateInvalid):
		return fiber.StatusUnprocessableEntity
	// User errors
	case errors.Is(err, user.ErrUserNotFound):
		return fiber.StatusNotFound
	case errors.Is(err, user.ErrUserUnauthorized):
		return fiber.StatusUnauthorized
	default:
		return fiber.StatusInternalServerError
	}
}
