package account

import (
	"encoding/json"
	"net/http"

	"github.com/amirasaad/fintech/pkg/processor"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
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
func StripeWebhookHandler(eventProcessor processor.EventProcessor, signingSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		const MaxBodyBytes = int64(65536)
		c.Request().Body()
		request := c.Request()
		body := request.Body()
		if int64(len(body)) > MaxBodyBytes {
			return c.Status(http.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "Request too large"})
		}

		// Stripe sends the signature in this header
		sigHeader := string(request.Header.Peek("Stripe-Signature"))
		if sigHeader == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Missing Stripe-Signature header"})
		}

		// Verify and parse the event
		event, err := webhook.ConstructEvent(body, sigHeader, signingSecret)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Webhook signature verification failed", "details": err.Error()})
		}

		var paymentID, status string
		// Handle only payment_intent events for now
		switch event.Type {
		case "payment_intent.succeeded":
			var pi stripe.PaymentIntent
			if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse payment_intent"})
			}
			paymentID = pi.ID
			status = "completed"
		case "payment_intent.payment_failed":
			var pi stripe.PaymentIntent
			if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse payment_intent"})
			}
			paymentID = pi.ID
			status = "failed"
		default:
			// Ignore other event types
			return c.SendStatus(http.StatusNoContent)
		}

		if paymentID == "" || status == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Missing payment ID or status"})
		}

		payEvent := processor.Event{
			Provider:  "stripe",
			PaymentID: paymentID,
			Status:    status,
			RawEvent:  event,
		}
		if err := eventProcessor.ProcessEvent(payEvent); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(http.StatusOK)
	}
}
