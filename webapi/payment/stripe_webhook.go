package payment

import (
	"fmt"

	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/gofiber/fiber/v2"
)

// StripeWebhookHandler handles incoming Stripe webhook events
func StripeWebhookHandler(
	paymentProvider provider.Payment,
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
		_, err := paymentProvider.HandleWebhook(c.Context(), payload, signature)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Error processing webhook: %v", err),
			})
		}

		// Return a 200 response to acknowledge receipt of the event
		return c.SendStatus(fiber.StatusOK)
	}
}

// StripeWebhookRoutes sets up the Stripe webhook routes
func StripeWebhookRoutes(
	app *fiber.App,
	paymentProvider provider.Payment,
) {
	// Webhook endpoint for Stripe events
	app.Post("/api/v1/webhooks/stripe", StripeWebhookHandler(paymentProvider))
}
