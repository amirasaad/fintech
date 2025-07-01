package handler

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
)

func ErrorToStatusCode(err error) int {
	switch err {
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
