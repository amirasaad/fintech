package app

import (
	"errors"
	"strings"

	"github.com/amirasaad/fintech/pkg/eventbus"
	handleraccount "github.com/amirasaad/fintech/pkg/handler/account"
	"github.com/amirasaad/fintech/pkg/processor"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	currencysvc "github.com/amirasaad/fintech/pkg/service/currency"
	usersvc "github.com/amirasaad/fintech/pkg/service/user"
	"github.com/amirasaad/fintech/webapi/account"
	"github.com/amirasaad/fintech/webapi/auth"
	"github.com/amirasaad/fintech/webapi/common"
	currencywebapi "github.com/amirasaad/fintech/webapi/currency"
	"github.com/amirasaad/fintech/webapi/user"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/swagger"

	"github.com/amirasaad/fintech/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	_ "github.com/amirasaad/fintech/cmd/server/swagger"
)

// New builds all services, registers event handlers, and returns the Fiber app.
func New(deps config.Deps) *fiber.App {
	// Build services
	accountSvc := accountsvc.NewService(deps)
	userSvc := usersvc.NewService(deps)
	authStrategy := authsvc.NewJWTAuthStrategy(deps.Uow, deps.Config.Jwt, deps.Logger)
	authSvc := authsvc.NewAuthService(deps.Uow, authStrategy, deps.Logger)
	currencySvc := currencysvc.NewCurrencyService(deps.CurrencyRegistry, deps.Logger)

	// Create a new context-aware event bus
	bus := eventbus.NewSimpleEventBus()

	// Register account validation flow handlers
	bus.Subscribe("AccountQuerySucceededEvent", handleraccount.AccountValidationHandler(bus, deps.Logger))
	bus.Subscribe("AccountValidatedEvent", handleraccount.MoneyCreationHandler(bus))

	// Register event-driven deposit workflow handlers
	bus.Subscribe("DepositRequestedEvent", handleraccount.DepositValidationHandler(bus))
	bus.Subscribe("DepositValidatedEvent", handleraccount.MoneyCreationHandler(bus))
	bus.Subscribe("MoneyCreatedEvent", handleraccount.DepositPersistenceHandler(bus))

	// Register event-driven withdraw workflow handlers
	bus.Subscribe("WithdrawRequestedEvent", handleraccount.WithdrawValidationHandler(bus))
	// TODO: Add WithdrawDomainOpHandler and WithdrawPersistenceHandler when implemented

	// Register event-driven transfer workflow handlers
	bus.Subscribe("TransferRequestedEvent", handleraccount.TransferValidationHandler(bus))
	bus.Subscribe("TransferValidatedEvent", handleraccount.TransferDomainOpHandler(bus /* TODO: inject TransferDomainOperator */, nil))
	bus.Subscribe("TransferDomainOpDoneEvent", handleraccount.TransferPersistenceHandler(bus /* TODO: inject TransferPersistenceAdapter */, nil))
	// Add more as you implement them

	// TODO: Create and register query handlers with a query bus or expose in API layer
	// Example: getAccountQueryHandler := handleraccount.GetAccountQueryHandler(deps.Uow, bus)

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
			return common.ProblemDetailsJSON(c, "Too Many Requests", errors.New("rate limit exceeded"), fiber.StatusTooManyRequests)
		},
	}))
	app.Use(recover.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("App is working! 🚀")
	})

	// Payment event processor for Stripe webhooks
	paymentEventProcessor := processor.NewDefaultPaymentEventProcessor(accountSvc)
	stripeSigningSecret := deps.Config.PaymentProviders.Stripe.SigningSecret
	app.Post("/payments/stripe/webhook", account.StripeWebhookHandler(paymentEventProcessor, stripeSigningSecret))

	app.Post("/webhook/payment", account.PaymentWebhookHandler(accountSvc))

	account.Routes(app, accountSvc, authSvc, deps.Config)
	user.Routes(app, userSvc, authSvc, deps.Config)
	auth.Routes(app, authSvc)
	currencywebapi.Routes(app, currencySvc, authSvc, deps.Config)
	return app
}
