package app

import (
	"errors"
	"strings"

	"github.com/amirasaad/fintech/pkg/handler/conversion"

	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	transferhandler "github.com/amirasaad/fintech/pkg/handler/account/transfer"
	withdrawhandler "github.com/amirasaad/fintech/pkg/handler/account/withdraw"
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
	bus := deps.EventBus

	// ============================================================================
	// EVENT HANDLER REGISTRATION - FINAL EVENT-DRIVEN ARCHITECTURE
	// ============================================================================

	// 1. GENERIC CONVERSION HANDLER (reusable across all operations)
	// Handles ConversionRequestedEvent only - ConversionDoneEvent is handled by business-specific handlers
	bus.Subscribe("ConversionRequestedEvent", conversion.Handler(bus, deps.CurrencyConverter, deps.Logger))

	// 2. DEPOSIT FLOW HANDLERS
	// Validation → Persistence → Conversion → Business Validation → Payment → [Conversion Persistence + Payment Persistence]
	bus.Subscribe("DepositRequestedEvent", deposithandler.ValidationHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("DepositValidatedEvent", deposithandler.PersistenceHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("DepositConversionDoneEvent", deposithandler.ConversionDoneHandler(bus, deps.Uow, deps.PaymentProvider, deps.Logger))
	bus.Subscribe("DepositConversionDoneEvent", deposithandler.ConversionPersistenceHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("PaymentInitiatedEvent", deposithandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))

	// 3. WITHDRAW FLOW HANDLERS
	// Validation → Persistence → Conversion → Business Validation → Payment → [Conversion Persistence + Payment Persistence]
	bus.Subscribe("WithdrawRequestedEvent", withdrawhandler.WithdrawValidationHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("WithdrawValidatedEvent", withdrawhandler.WithdrawPersistenceHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("WithdrawConversionDoneEvent", withdrawhandler.ConversionDoneHandler(bus, deps.Uow, deps.PaymentProvider, deps.Logger))
	bus.Subscribe("WithdrawConversionDoneEvent", withdrawhandler.ConversionPersistenceHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("PaymentInitiatedEvent", withdrawhandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))

	// 4. TRANSFER FLOW HANDLERS
	// Validation → Initial Persistence → Conversion → Business Validation → Domain Op → Final Persistence → Conversion Persistence
	bus.Subscribe("TransferRequestedEvent", transferhandler.TransferValidationHandler(bus, deps.Logger))
	bus.Subscribe("TransferValidatedEvent", transferhandler.InitialPersistenceHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("TransferConversionDoneEvent", transferhandler.ConversionDoneHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("TransferConversionDoneEvent", transferhandler.ConversionPersistenceHandler(bus, deps.Uow, deps.Logger))
	bus.Subscribe("TransferConversionDoneEvent", transferhandler.TransferDomainOpHandler(bus, nil))
	bus.Subscribe("TransferDomainOpDoneEvent", transferhandler.TransferPersistenceHandler(bus, deps.Uow, deps.Logger))

	// 5. PAYMENT HANDLERS (for payment completion)
	// TODO: Implement payment handlers when needed
	// bus.Subscribe("PaymentInitiatedEvent", account.PaymentInitiationHandler(bus, deps.Logger))
	// bus.Subscribe("PaymentCompletedEvent", account.PaymentCompletedHandler(bus, deps.Logger))

	// ============================================================================
	// LEGACY HANDLER REGISTRATION (for backward compatibility)
	// ============================================================================

	// Legacy conversion events
	bus.Subscribe("ConversionRequested", conversion.Handler(bus, deps.CurrencyConverter, deps.Logger))

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
	stripeSigningSecret := deps.Config.PaymentProviders.Stripe.SigningSecret
	app.Post("/payments/stripe/webhook", account.StripeWebhookHandler(bus, stripeSigningSecret))

	app.Post("/webhook/payment", account.PaymentWebhookHandler(accountSvc))

	account.Routes(app, accountSvc, authSvc, deps.Config)
	user.Routes(app, userSvc, authSvc, deps.Config)
	auth.Routes(app, authSvc)
	currencywebapi.Routes(app, currencySvc, authSvc, deps.Config)
	return app
}
