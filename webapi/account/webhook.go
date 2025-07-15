package account

import (
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/gofiber/fiber/v2"
)

// PaymentWebhookRequest represents the payload for a payment webhook callback.
type PaymentWebhookRequest struct {
	PaymentID string `json:"payment_id" validate:"required"`
	Status    string `json:"status" validate:"required,oneof=completed failed"`
}

// PaymentWebhookHandler handles incoming payment webhook callbacks.
func PaymentWebhookHandler(svc *account.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req PaymentWebhookRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
		}
		if err := svc.UpdateTransactionStatusByPaymentID(req.PaymentID, req.Status); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(fiber.StatusOK)
	}
}
