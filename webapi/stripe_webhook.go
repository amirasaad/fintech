package webapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// StripeWebhookHandler handles incoming Stripe webhook events
func StripeWebhookHandler(
	paymentProvider provider.PaymentProvider,
	eventBus eventbus.Bus,
) fiber.Handler {
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

			// Get user ID and account ID from metadata if available
			userID, _ := uuid.Parse(event.Metadata["user_id"])
			accountID, _ := uuid.Parse(event.Metadata["account_id"])

			// Create flow event with correlation ID set to transaction ID
			flowEvent := events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "payment",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: transactionID,
				Timestamp:     time.Now().UTC(),
			}

			// Create payment initiated event
			paymentInitiated := events.NewPaymentInitiated(
				flowEvent,
				events.WithPaymentTransactionID(transactionID),
				events.WithInitiatedPaymentID(event.ID),
				events.WithInitiatedPaymentStatus(string(event.Status)),
			)

			// Emit appropriate event based on payment status
			switch event.Status {
			case provider.PaymentCompleted:
				// Emit payment completed event
				completedEvent := &events.PaymentCompleted{
					PaymentInitiated: *paymentInitiated,
				}
				if err := eventBus.Emit(c.Context(), completedEvent); err != nil {
					errMsg := fmt.Sprintf("Failed to emit payment completed event: %v", err)
					c.Locals("error", errMsg)
				}

			case provider.PaymentFailed:
				// Emit payment failed event
				failedEvent := &events.PaymentFailed{
					PaymentInitiated: *paymentInitiated,
					Reason:           "Payment failed",
				}
				if err := eventBus.Emit(c.Context(), failedEvent); err != nil {
					errMsg := fmt.Sprintf("Failed to emit payment failed event: %v", err)
					c.Locals("error", errMsg)
				}

			default:
				// For other statuses, just emit the payment processed event
				processedEvent := &events.PaymentProcessed{
					PaymentInitiated: *paymentInitiated,
				}
				if err := eventBus.Emit(c.Context(), processedEvent); err != nil {
					errMsg := fmt.Sprintf(
						"Failed to emit payment processed event: %v",
						err,
					)
					c.Locals("error", errMsg)
				}
			}
		}

		// Return a 200 response to acknowledge receipt of the event
		return c.SendStatus(fiber.StatusOK)
	}
}

// StripeWebhookRoutes sets up the Stripe webhook routes
func StripeWebhookRoutes(
	app *fiber.App,
	paymentProvider provider.PaymentProvider,
	eventBus eventbus.Bus,
) {
	// Webhook endpoint for Stripe events
	app.Post("/api/v1/webhooks/stripe", StripeWebhookHandler(paymentProvider, eventBus))
}
