package webapi

import (
	"encoding/json"
	"fmt"

	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// StripeWebhookHandler handles incoming Stripe webhook events
func StripeWebhookHandler(paymentProvider provider.PaymentProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the signature from the request headers
		signature := c.Get("Stripe-Signature")
		if signature == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing Stripe-Signature header",
			})
		}

		// Get the raw request body
		payload := c.Body()
		if len(payload) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Empty request body",
			})
		}

		// Process the webhook event
		event, err := paymentProvider.HandleWebhook(payload, signature)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Error processing webhook: %v", err),
			})
		}

		// If we have an event, process it
		if event != nil {
			// Log the event for debugging
			jsonEvent, _ := json.Marshal(event)
			c.Locals("stripe_event", jsonEvent)

			// Get the transaction ID from the event metadata
			transactionID, err := uuid.Parse(
				event.Metadata["transaction_id"],
			)
			if err != nil {
				// Log the error but continue processing
				c.Locals(
					"error",
					fmt.Sprintf(
						"Invalid transaction_id in webhook event: %v",
						err,
					),
				)
				return c.SendStatus(fiber.StatusOK)
			}

			// Update the payment record based on the event status
			switch event.Status {
			case provider.PaymentCompleted:
				// Update the payment record as completed
				updateParams := &provider.UpdatePaymentStatusParams{
					TransactionID: transactionID,
					PaymentID:     event.ID,
					Status:        provider.PaymentCompleted,
				}
				err = paymentProvider.UpdatePaymentStatus(
					c.Context(),
					updateParams,
				)
			case provider.PaymentFailed:
				// Update the payment record as failed
				updateParams := &provider.UpdatePaymentStatusParams{
					TransactionID: transactionID,
					PaymentID:     event.ID,
					Status:        provider.PaymentFailed,
				}
				err = paymentProvider.UpdatePaymentStatus(
					c.Context(),
					updateParams,
				)
			}

			if err != nil {
				c.Locals("error", fmt.Sprintf("Error updating payment status: %v", err))
			}
		}

		// Return a 200 response to acknowledge receipt of the event
		return c.SendStatus(fiber.StatusOK)
	}
}

// StripeWebhookRoutes sets up the Stripe webhook routes
func StripeWebhookRoutes(app *fiber.App, paymentProvider provider.PaymentProvider) {
	// Webhook endpoint for Stripe events
	app.Post("/api/v1/webhooks/stripe", StripeWebhookHandler(paymentProvider))
}
