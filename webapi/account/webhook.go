package account

import (
	"encoding/json"
	"net/http"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
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

// StripeWebhookHandler handles incoming Stripe webhook events.
func StripeWebhookHandler(eventBus eventbus.Bus, signingSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		const MaxBodyBytes = int64(65536)
		request := c.Request()
		body := request.Body()
		if int64(len(body)) > MaxBodyBytes {
			return c.SendStatus(http.StatusRequestEntityTooLarge)
		}

		event := stripe.Event{}
		if err := json.Unmarshal(body, &event); err != nil {
			return c.SendStatus(http.StatusBadRequest)
		}

		var paymentIntent stripe.PaymentIntent
		if event.Type == "payment_intent.succeeded" {
			if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
				return c.SendStatus(http.StatusBadRequest)
			}

			// Publish PaymentCompletedEvent to the event bus
			paymentEvent := events.NewPaymentCompletedEvent(
				uuid.Nil, // UserID is not available in this context
				uuid.Nil, // AccountID is not available in this context
				events.WithPaymentID(paymentIntent.ID),
			)
			if err := eventBus.Emit(c.Context(), paymentEvent); err != nil {
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		} else {
			// For now, ignore other event types
			return c.SendStatus(http.StatusOK)
		}
		return c.SendStatus(http.StatusOK)
	}
}
