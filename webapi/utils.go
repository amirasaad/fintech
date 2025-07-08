package webapi

import (
	"errors"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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

// ErrorResponseJSON returns a response following RFC 9457 Problem Details
func ErrorResponseJSON(
	c *fiber.Ctx,
	status int,
	title string,
	detail any,
) error {
	pd := ProblemDetails{
		Type:   "about:blank",
		Title:  title,
		Status: status,
	}
	if detail != nil {
		if s, ok := detail.(string); ok {
			pd.Detail = s
		} else {
			pd.Errors = detail
		}
	}
	pd.Instance = c.OriginalURL()
	c.Set(fiber.HeaderContentType, "application/problem+json")

	return c.Status(status).JSON(pd)
}

// ErrorToStatusCode maps domain errors to appropriate HTTP status codes.
func ErrorToStatusCode(err error) int {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return fiber.StatusNotFound
	case errors.Is(err, domain.ErrInvalidCurrencyCode):
		return fiber.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrDepositAmountExceedsMaxSafeInt):
		return fiber.StatusBadRequest
	case errors.Is(err, domain.ErrTransactionAmountMustBePositive):
		return fiber.StatusBadRequest
	case errors.Is(err, domain.ErrWithdrawalAmountMustBePositive):
		return fiber.StatusBadRequest
	case errors.Is(err, domain.ErrInsufficientFunds):
		return fiber.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrUserUnauthorized):
		return fiber.StatusUnauthorized
	default:
		return fiber.StatusInternalServerError
	}
}

// BindAndValidate parses the request body and validates it using go-playground/validator.
// Returns a pointer to the struct (populated), or writes an error response and returns nil.
func BindAndValidate[T any](c *fiber.Ctx) (*T, error) {
	var input T
	if err := c.BodyParser(&input); err != nil {
		ErrorResponseJSON(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
		return nil, err
	}
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		ErrorResponseJSON(c, fiber.StatusBadRequest, "Validation failed", err.Error())
		return nil, err
	}
	return &input, nil
}
