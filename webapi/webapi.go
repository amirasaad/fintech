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

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/app"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	currencysvc "github.com/amirasaad/fintech/pkg/service/currency"
	usersvc "github.com/amirasaad/fintech/pkg/service/user"
	accountweb "github.com/amirasaad/fintech/webapi/account"
	authweb "github.com/amirasaad/fintech/webapi/auth"
	"github.com/amirasaad/fintech/webapi/common"
	currencyweb "github.com/amirasaad/fintech/webapi/currency"
	"github.com/amirasaad/fintech/webapi/payment"
	userweb "github.com/amirasaad/fintech/webapi/user"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
)

// SetupApp Initialize Fiber with custom configuration
func SetupApp(deps config.Deps) *fiber.App {
	// Setup Bus
	// Configure the event bus with all handlers
	app.SetupBus(app.Dependencies{
		Bus:               deps.EventBus,
		Uow:               deps.Uow,
		Logger:            deps.Logger,
		CurrencyConverter: deps.CurrencyConverter,
		PaymentProvider:   deps.PaymentProvider,
	})

	// Build services
	accountSvc := accountsvc.New(deps.EventBus, deps.Uow, deps.Logger)
	userSvc := usersvc.New(deps.Uow, deps.Logger)
	authStrategy := authsvc.NewWithJWT(deps.Uow, deps.Config.Jwt, deps.Logger)
	authSvc := authsvc.New(deps.Uow, authStrategy, deps.Logger)
	currencySvc := currencysvc.NewService(deps.CurrencyRegistry, deps.Logger)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return common.ProblemDetailsJSON(c, "Internal Server Error", err)
		},
	})
	app.Get("/swagger/*", swagger.New(swagger.Config{
		TryItOutEnabled:      true,
		WithCredentials:      true,
		PersistAuthorization: true,
		OAuth2RedirectUrl:    "/auth/login",
	}))

	// Configure rate limiting middleware
	// Uses X-Forwarded-For header when behind a proxy
	// Falls back to X-Real-IP or direct IP if needed
	app.Use(limiter.New(limiter.Config{
		Max:        deps.Config.RateLimit.MaxRequests,
		Expiration: deps.Config.RateLimit.Window,
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
	app.Use(recover.New())

	// Health check endpoint
	app.Get(
		"/",
		func(c *fiber.Ctx) error {
			return c.SendString("FinTech API is running! 🚀")
		},
	)

	// Payment event processor for Stripe webhooks
	app.Post(
		"/api/v1/webhooks/stripe",
		payment.StripeWebhookHandler(deps.PaymentProvider),
	)

	accountweb.Routes(app, accountSvc, authSvc, deps.Config)
	userweb.Routes(app, userSvc, authSvc, deps.Config)
	authweb.Routes(app, authSvc)
	currencyweb.Routes(app, currencySvc, authSvc, deps.Config)
	return app
}
