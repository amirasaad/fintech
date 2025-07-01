// The `package handler` is defining a Go package that likely contains functions and methods related to
// handling HTTP requests and responses in a web application. The code snippet provided includes a
// function `ErrorToStatusCode` that takes an error as input and returns an HTTP status code based on
// the type of error. This function is used to map domain-specific errors to appropriate HTTP status
// codes for handling errors in the web application.
package handler

import (
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/gofiber/fiber/v2"
)

// The function `ErrorToStatusCode` maps specific domain errors to corresponding HTTP status codes in a
// Go application.
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
