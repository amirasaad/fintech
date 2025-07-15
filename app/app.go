package app

import (
	"context"
	"errors"
	"strings"

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

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/handler"

	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
)

// New builds all services, registers event handlers, and returns the Fiber app.
func New(deps config.Deps) *fiber.App {
	// Build services
	accountSvc := accountsvc.NewService(deps)
	userSvc := usersvc.NewService(deps)
	authStrategy := authsvc.NewJWTAuthStrategy(deps.Uow, deps.Config.Jwt, deps.Logger)
	authSvc := authsvc.NewAuthService(deps.Uow, authStrategy, deps.Logger)
	currencySvc := currencysvc.NewCurrencyService(deps.CurrencyRegistry, deps.Logger)

	// Register event handlers (example for DepositRequestedEvent)
	accountChain := handler.NewAccountChain(deps.Uow, deps.CurrencyConverter, deps.Logger)
	deps.EventBus.Subscribe("DepositRequestedEvent", func(e domain.Event) {
		// Use type assertion with ok check
		if evt, ok := e.(accountdomain.DepositRequestedEvent); ok {
			userID := uuid.MustParse(evt.UserID)
			accountID := uuid.MustParse(evt.AccountID)
			amount := evt.Amount
			currencyCode := currency.Code(evt.Currency)
			moneySource := string(evt.Source)
			_, err := accountChain.Deposit(context.Background(), userID, accountID, amount, currencyCode, moneySource)
			if err != nil {
				deps.Logger.Error("Deposit event handler failed", "error", err)
			}
		} else {
			deps.Logger.Error("event type assertion failed", "event", e)
		}
	})
	deps.EventBus.Subscribe("WithdrawRequestedEvent", func(e domain.Event) {
		if evt, ok := e.(accountdomain.WithdrawRequestedEvent); ok {
			userID := uuid.MustParse(evt.UserID)
			accountID := uuid.MustParse(evt.AccountID)
			amount := evt.Amount
			currencyCode := currency.Code(evt.Currency)
			externalTarget := evt.Target
			_, err := accountChain.WithdrawExternal(context.Background(), userID, accountID, amount, currencyCode, handler.ExternalTarget(externalTarget))
			if err != nil {
				deps.Logger.Error("Withdraw event handler failed", "error", err)
			}
		} else {
			deps.Logger.Error("event type assertion failed", "event", e)
		}
	})
	deps.EventBus.Subscribe("TransferRequestedEvent", func(e domain.Event) {
		if evt, ok := e.(accountdomain.TransferRequestedEvent); ok {
			senderUserID := evt.SenderUserID
			receiverUserID := evt.ReceiverUserID
			sourceAccID := evt.SourceAccountID
			destAccID := evt.DestAccountID
			amount := evt.Amount
			currencyCode := currency.Code(evt.Currency)
			_, err := accountChain.Transfer(context.Background(), senderUserID, receiverUserID, sourceAccID, destAccID, amount, currencyCode)
			if err != nil {
				deps.Logger.Error("Deposit event handler failed", "error", err)
			}
		} else {
			deps.Logger.Error("event type assertion failed", "event", e)
		}
	})

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

	app.Post("/webhook/payment", account.PaymentWebhookHandler(accountSvc))

	account.Routes(app, accountSvc, authSvc, deps.Config)
	user.Routes(app, userSvc, authSvc, deps.Config)
	auth.Routes(app, authSvc)
	currencywebapi.Routes(app, currencySvc, authSvc, deps.Config)
	return app
}
