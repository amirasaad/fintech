// Package app initializes and configures the FinTech application's core components.
// It sets up the Fiber web framework, registers all event handlers, and configures
// the application's routing and middleware. The package follows a clean architecture
// with clear separation of concerns between different business domains.
package app

import (
	"errors"
	"strings"

	"github.com/amirasaad/fintech/pkg/eventbus"

	"github.com/amirasaad/fintech/pkg/handler/conversion"
	paymenthandler "github.com/amirasaad/fintech/pkg/handler/payment"

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
	accountSvc := accountsvc.NewService(deps.EventBus, deps.Uow, deps.Logger)
	userSvc := usersvc.NewService(deps.Uow, deps.Logger)
	authStrategy := authsvc.NewJWTAuthStrategy(deps.Uow, deps.Config.Jwt, deps.Logger)
	authSvc := authsvc.NewAuthService(deps.Uow, authStrategy, deps.Logger)
	currencySvc := currencysvc.NewCurrencyService(deps.CurrencyRegistry, deps.Logger)

	bus := SetupBus(deps)
	app := SetupApp(deps, bus, accountSvc, authSvc, userSvc, currencySvc)
	return app
}

// SetupApp Initialize Fiber with custom configuration
func SetupApp(deps config.Deps, bus eventbus.Bus, accountSvc *accountsvc.Service, authSvc *authsvc.AuthService, userSvc *usersvc.Service, currencySvc *currencysvc.CurrencyService) *fiber.App {
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
			return common.ProblemDetailsJSON(c, "Too Many Requests", errors.New("rate limit exceeded"), fiber.StatusTooManyRequests)
		},
	}))
	app.Use(recover.New())

	// Health check endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("FinTech API is running! 🚀")
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

// SetupBus Registers each event with its delegated handler
//
// 📝 ARCHITECTURE NOTES
//
// Key Design Principles:
// - Single Responsibility: Each handler does exactly one thing
// - Event Sourcing: Business processes are modeled as a series of events
// - CQRS: Commands (write operations) are separated from queries (reads)
// - Domain-Driven Design: Core business logic is encapsulated in domain models
//
// Event Flow Patterns:
// 1. Validation → Persistence → Business Logic → External Integration
// 2. Fail Fast: Validation happens before any state changes
// 3. Idempotency: Handlers are designed to be safely retryable
//
// Security:
// - Rate limiting is applied globally
// - Authentication is required for all endpoints (except public ones)
// - Input validation happens at the API boundary
// ============================================================================
func SetupBus(deps config.Deps) eventbus.Bus {
	// Create a new context-aware event bus
	bus := deps.EventBus

	// ============================================================================
	// 🧩 EVENT HANDLER REGISTRATION – CLEAN, SINGLE-PASS EVENT-DRIVEN ARCHITECTURE
	//
	// The application uses a single-pass event-driven architecture where:
	// - Each event represents a discrete step in a business process
	// - Handlers are responsible for exactly one task
	// - Events flow in one direction through the system
	// - No handler emits an event that would trigger itself (prevents cycles)
	// - Business validation happens after conversion to account currency
	// ============================================================================
	// 1️⃣ GENERIC CONVERSION HANDLER
	// Handles all ConversionRequestedEvent by delegating to a flow-specific factory.
	conversionFactories := map[string]conversion.EventFactory{
		"deposit":  &conversion.DepositEventFactory{},
		"withdraw": &conversion.WithdrawEventFactory{},
		"transfer": &conversion.TransferEventFactory{},
	}

	bus.Register("ConversionRequestedEvent", conversion.Handler(bus, deps.CurrencyConverter, deps.Logger, conversionFactories))
	bus.Register("ConversionDoneEvent", conversion.Persistence(deps.Uow, deps.Logger))
	// 2️⃣ DEPOSIT FLOW
	// User request → Initial Validation → Persistence → Conversion → Business Validation → Payment Initiation → Payment Persistence

	// a. Initial validation of deposit request
	bus.Register("DepositRequestedEvent", deposithandler.Validation(bus, deps.Uow, deps.Logger))
	// b. Persist transaction after validation
	bus.Register("DepositValidatedEvent", deposithandler.Persistence(bus, deps.Uow, deps.Logger))
	// c. Business validation after conversion (in account currency)
	bus.Register("DepositBusinessValidationEvent", deposithandler.BusinessValidation(bus, deps.Uow, deps.Logger))
	// d. Deposit Business Validated Event
	bus.Register("DepositBusinessValidatedEvent", paymenthandler.Initiation(bus, deps.PaymentProvider, deps.Logger))

	// 3️⃣ WITHDRAW FLOW
	// User request → Initial Validation → Persistence → Conversion → Business Validation → Payment Initiation → Payment Persistence

	// a. Initial validation of withdraw request
	bus.Register("WithdrawRequestedEvent", withdrawhandler.Validation(bus, deps.Uow, deps.Logger))
	// b. Persist transaction after validation
	bus.Register("WithdrawValidatedEvent", withdrawhandler.Persistence(bus, deps.Uow, deps.Logger))
	// c. Business validation after conversion (in account currency)
	bus.Register("WithdrawBusinessValidationEvent", withdrawhandler.BusinessValidation(bus, deps.Uow, deps.Logger))
	// d. Withdraw Business Validated Event
	bus.Register("WithdrawBusinessValidatedEvent", paymenthandler.Initiation(bus, deps.PaymentProvider, deps.Logger))

	// Payment workflow
	// Payment Initiation → Payment Persistence
	// (External via Webhook) Payment Completion -> Payment Completed
	bus.Register("PaymentInitiationEvent", paymenthandler.Initiation(bus, deps.PaymentProvider, deps.Logger))
	bus.Register("PaymentInitiatedEvent", paymenthandler.Persistence(bus, deps.Uow, deps.Logger))
	bus.Register("PaymentCompletedEvent", paymenthandler.Completed(bus, deps.Uow, deps.Logger))

	// 4️⃣ TRANSFER FLOW
	// User request → Initial Validation → Initial Persistence → Conversion → Business Validation → Final Persistence

	// a. Initial validation of transfer request
	bus.Register("TransferRequestedEvent", transferhandler.Validation(bus, deps.Logger))
	// b. Initial persistence after validation
	bus.Register("TransferValidatedEvent", transferhandler.InitialPersistence(bus, deps.Uow, deps.Logger))
	// c. Business validation after conversion
	bus.Register("TransferBusinessValidationEvent", transferhandler.BusinessValidation(bus, deps.Uow, deps.Logger))
	// d. Final persistence after domain operation
	bus.Register("TransferDomainOpDoneEvent", transferhandler.Persistence(bus, deps.Uow, deps.Logger))

	return bus
}
