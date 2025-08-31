// Package webapi provides HTTP handlers and API endpoints for the fintech application.
// It is organized into sub-packages for different domains:
// - account: Account and transaction endpoints
// - auth: Authentication endpoints
// - user: User management endpoints
// - currency: Currency and exchange rate endpoints
package webapi

import (
	"errors"
	"strings"

	"github.com/amirasaad/fintech/pkg/app"
	accountweb "github.com/amirasaad/fintech/webapi/account"
	authweb "github.com/amirasaad/fintech/webapi/auth"
	checkoutweb "github.com/amirasaad/fintech/webapi/checkout"
	"github.com/amirasaad/fintech/webapi/common"
	currencyweb "github.com/amirasaad/fintech/webapi/currency"
	"github.com/amirasaad/fintech/webapi/payment"
	userweb "github.com/amirasaad/fintech/webapi/user"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/gofiber/swagger"
)

// SetupApp Initialize Fiber with custom configuration
func SetupApp(app *app.App) *fiber.App {
	accountSvc := app.AccountService
	userSvc := app.UserService
	authSvc := app.AuthService
	currencySvc := app.CurrencyService
	checkoutSvc := app.CheckoutService

	fiberApp := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return common.ProblemDetailsJSON(c, "Internal Server Error", err)
		},
	})
	fiberApp.Get("/swagger/*", swagger.New(swagger.Config{
		TryItOutEnabled:      true,
		WithCredentials:      true,
		PersistAuthorization: true,
		OAuth2RedirectUrl:    "/auth/login",
	}))

	// Configure rate limiting middleware
	// Uses X-Forwarded-For header when behind a proxy
	// Falls back to X-Real-IP or direct IP if needed
	fiberApp.Use(limiter.New(limiter.Config{
		Max:        app.Config.RateLimit.MaxRequests,
		Expiration: app.Config.RateLimit.Window,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use X-Forwarded-For header if available (for load balancers/proxies)
			// Fall back to X-Real-IP, then to direct IP
			if forwardedFor := c.Get("X-Forwarded-For"); forwardedFor != "" {
				// Take the first IP in the chain
				if commaIndex := strings.Index(forwardedFor, ","); commaIndex != -1 {
					return strings.TrimSpace(forwardedFor[:commaIndex])
				}
				return strings.TrimSpace(forwardedFor)
			}
			if realIP := c.Get("X-Real-IP"); realIP != "" {
				return realIP
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return common.ProblemDetailsJSON(
				c,
				"Too Many Requests",
				errors.New("rate limit exceeded"),
				fiber.StatusTooManyRequests,
			)
		},
	}))
	fiberApp.Use(recover.New())
	fiberApp.Use(logger.New())

	// Health check endpoint
	fiberApp.Get(
		"/",
		func(c *fiber.Ctx) error {
			return c.SendString("FinTech API is running! 🚀")
		},
	)

	// Debug endpoint to list all routes
	fiberApp.Get("/debug/routes", func(c *fiber.Ctx) error {
		routes := fiberApp.GetRoutes()
		var routeList []map[string]interface{}
		for _, route := range routes {
			if route.Path != "" {
				routeList = append(routeList, map[string]interface{}{
					"method": route.Method,
					"path":   route.Path,
				})
			}
		}
		return c.JSON(routeList)
	})

	// Payment event processor for Stripe webhooks
	fiberApp.Post(
		"/stripe/webhooks",
		payment.StripeWebhookHandler(app.Deps.PaymentProvider),
	)

	// Initialize account routes which include Stripe Connect routes
	accountweb.Routes(fiberApp, accountSvc, authSvc, app.StripeConnectService, app.Config)
	userweb.Routes(fiberApp, userSvc, authSvc, app.Config)
	authweb.Routes(fiberApp, authSvc)
	currencyweb.Routes(fiberApp, currencySvc, authSvc, app.Config)
	checkoutweb.Routes(fiberApp, checkoutSvc, authSvc, app.Config)
	return fiberApp
}
