package webapi

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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
	default:
		return fiber.StatusInternalServerError
	}
}

func GetCurrentUserId(c *fiber.Ctx) (uuid.UUID, error) {
	tokenVal := c.Locals("user")
	log.Infof("Token value: %v type %T", tokenVal, tokenVal)
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

func CheckUserAccountOwnership(c *fiber.Ctx, account *domain.Account) (bool, error) {
	userID, err := GetCurrentUserId(c)
	if err != nil {
		return false, err
	}
	if userID != account.UserID {
		log.Errorf("User ID from token does not match account ID")
		return false, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user ID does not match account ID"})
	}
	return true, nil
}
