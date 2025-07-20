package app

import (
	"errors"
	"fmt"
	"strings"

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
	accountSvc := accountsvc.NewService(deps)
	userSvc := usersvc.NewService(deps)
	authStrategy := authsvc.NewJWTAuthStrategy(deps.Uow, deps.Config.Jwt, deps.Logger)
	authSvc := authsvc.NewAuthService(deps.Uow, authStrategy, deps.Logger)
	currencySvc := currencysvc.NewCurrencyService(deps.CurrencyRegistry, deps.Logger)

	// Create a new context-aware event bus
	bus := deps.EventBus

	// ============================================================================
	// 🧩 EVENT HANDLER REGISTRATION – CLEAN, SINGLE-PASS EVENT-DRIVEN ARCHITECTURE
	// ============================================================================

	// 1️⃣ GENERIC CONVERSION HANDLER
	// Handles all ConversionRequestedEvent for any operation (deposit, withdraw, transfer)
	fmt.Println("Registering handler for event type:", "ConversionRequestedEvent")
	bus.Register("ConversionRequestedEvent", conversion.Handler(bus, deps.CurrencyConverter, deps.Logger))

	// 2️⃣ DEPOSIT FLOW
	// User request → Initial Validation → Persistence → Conversion → Business Validation → Payment Initiation → Payment Persistence

	// a. Initial validation of deposit request
	bus.Register("DepositRequestedEvent", deposithandler.ValidationHandler(bus, deps.Uow, deps.Logger))
	// b. Persist transaction after validation
	bus.Register("DepositValidatedEvent", deposithandler.PersistenceHandler(bus, deps.Uow, deps.Logger))
	// c. Conversion done: persist conversion data
	bus.Register("DepositConversionDoneEvent", deposithandler.ConversionPersistenceHandler(bus, deps.Uow, deps.Logger))
	// d. Business validation after conversion (in account currency)
	bus.Register("DepositConversionDoneEvent", deposithandler.BusinessValidationHandler(bus, deps.Logger))
	// e. Payment initiation (only after business validation)
	bus.Register("DepositBusinessValidatedEvent", paymenthandler.PaymentInitiationHandler(bus, deps.PaymentProvider, deps.Logger))
	// f. Payment persistence (store payment ID, etc.)
	bus.Register("PaymentInitiatedEvent", paymenthandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))

	// 3️⃣ WITHDRAW FLOW
	// User request → Initial Validation → Persistence → Conversion → Business Validation → Payment Initiation → Payment Persistence

	// a. Initial validation of withdraw request
	bus.Register("WithdrawRequestedEvent", withdrawhandler.WithdrawValidationHandler(bus, deps.Uow, deps.Logger))
	// b. Persist transaction after validation
	bus.Register("WithdrawValidatedEvent", withdrawhandler.WithdrawPersistenceHandler(bus, deps.Uow, deps.Logger))
	// c. Conversion done: persist conversion data
	bus.Register("WithdrawConversionDoneEvent", withdrawhandler.ConversionPersistenceHandler(bus, deps.Uow, deps.Logger))
	// d. Business validation after conversion (in account currency)
	bus.Register("WithdrawConversionDoneEvent", withdrawhandler.BusinessValidationHandler(bus, deps.Logger))
	// e. Payment initiation (after business validation or directly after conversion if no extra validation step)
	bus.Register("WithdrawValidatedEvent", paymenthandler.PaymentInitiationHandler(bus, deps.PaymentProvider, deps.Logger))
	// f. Payment persistence
	bus.Register("PaymentInitiatedEvent", paymenthandler.PaymentPersistenceHandler(bus, deps.Uow, deps.Logger))

	// 4️⃣ TRANSFER FLOW
	// User request → Initial Validation → Initial Persistence → Conversion → Domain Operation → Final Persistence

	// a. Initial validation of transfer request
	bus.Register("TransferRequestedEvent", transferhandler.TransferValidationHandler(bus, deps.Logger))
	// b. Initial persistence after validation
	bus.Register("TransferValidatedEvent", transferhandler.InitialPersistenceHandler(bus, deps.Uow, deps.Logger))
	// c. Conversion done: persist conversion data
	bus.Register("TransferConversionDoneEvent", transferhandler.ConversionPersistenceHandler(bus, deps.Uow, deps.Logger))
	// d. Domain operation (move funds between accounts)
	bus.Register("TransferConversionDoneEvent", transferhandler.TransferDomainOpHandler(bus, nil))
	// e. Final persistence after domain operation
	bus.Register("TransferDomainOpDoneEvent", transferhandler.TransferPersistenceHandler(bus, deps.Uow, deps.Logger))
	// f. Business validation after conversion (in account currency)
	bus.Register("TransferConversionDoneEvent", transferhandler.BusinessValidationHandler(bus, deps.Logger))

	// Conversion done handlers (flow-specific)
	bus.Register("DepositConversionDoneEvent", deposithandler.ConversionDoneHandler(bus, deps.Uow, deps.Logger))
	bus.Register("WithdrawConversionDoneEvent", withdrawhandler.ConversionDoneHandler(bus, deps.Uow, deps.Logger))
	bus.Register("TransferConversionDoneEvent", transferhandler.ConversionDoneHandler(bus, deps.Uow, deps.Logger))

	// ============================================================================
	// 📝 DOCUMENTATION
	// - Each handler is responsible for a single step in the workflow (SRP).
	// - Payment initiation is only triggered after business validation (not for internal transfers).
	// - Conversion handlers are generic and reusable across flows.
	// - No handler emits an event that would trigger itself or create a cycle (prevents infinite loops).
	// - Add/extend business validation handlers for withdraw/transfer as needed for your domain.
	// ============================================================================

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
