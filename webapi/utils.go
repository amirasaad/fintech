package webapi

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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
func ErrorResponseJSON(c *fiber.Ctx, status int, title string, detail any) error {
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
	return c.Status(status).JSON(pd)
}

// ErrorToStatusCode maps domain errors to appropriate HTTP status codes.
func ErrorToStatusCode(err error) int {
	switch err {
	case domain.ErrAccountNotFound:
		return fiber.StatusNotFound
	case domain.ErrDepositAmountExceedsMaxSafeInt:
		return fiber.StatusBadRequest
	case domain.ErrTransactionAmountMustBePositive:
		return fiber.StatusBadRequest
	case domain.ErrWithdrawalAmountMustBePositive:
		return fiber.StatusBadRequest
	case domain.ErrInsufficientFunds:
		return fiber.StatusUnprocessableEntity
	case domain.ErrUserUnauthorized:
		return fiber.StatusUnauthorized
	default:
		return fiber.StatusInternalServerError
	}
}

func GetCurrentUserId(c *fiber.Ctx) (uuid.UUID, error) {
	tokenVal := c.Locals("user")
	if tokenVal == nil {
		log.Error("Missing or invalid token")
		return uuid.Nil, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid token"})
	}
	token, ok := tokenVal.(*jwt.Token)
	log.Infof("Token type: %T", token)
	if !ok {
		log.Errorf("Invalid token type %s", ok)
		return uuid.Nil, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token type"})
	}
	claims := token.Claims.(jwt.MapClaims)
	userID, err := uuid.Parse(claims["user_id"].(string))
	if err != nil {
		log.Errorf("Failed to parse user ID from token: %v", err)
		return uuid.Nil, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user ID"})
	}

	return userID, nil
}
