package account

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/amirasaad/fintech/pkg/processor"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/gofiber/fiber/v2"
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
func StripeWebhookHandler(eventProcessor processor.EventProcessor, signingSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		const MaxBodyBytes = int64(65536)
		c.Request().Body()
		request := c.Request()
		body := request.Body()
		if int64(len(body)) > MaxBodyBytes {
			return c.SendStatus(http.StatusRequestEntityTooLarge)
		}

		event := stripe.Event{}

		if err := json.Unmarshal(body, &event); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse webhook body json: %v\n", err.Error())
			return c.SendStatus(http.StatusBadRequest)
		}

		// Unmarshal the event data into an appropriate struct depending on its Type
		var paymentMethod stripe.PaymentMethod
		var paymentIntent stripe.PaymentIntent
		switch event.Type {
		case "payment_intent.succeeded":
			err := json.Unmarshal(event.Data.Raw, &paymentIntent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				return c.SendStatus(http.StatusBadRequest)
			}
			fmt.Printf("payment_intent.succeeded, %v", paymentIntent)

		case "payment_method.attached":
			err := json.Unmarshal(event.Data.Raw, &paymentMethod)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				return c.SendStatus(http.StatusBadRequest)
			}
			fmt.Printf("payment_method.attached %v", paymentMethod)

		default:
			fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		}

		payEvent := processor.Event{
			Provider:  "stripe",
			PaymentID: paymentIntent.ID,
			Status:    string(paymentIntent.Status),
			RawEvent:  event,
		}
		if err := eventProcessor.ProcessEvent(payEvent); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(http.StatusOK)
	}
}
