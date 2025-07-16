package account

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// AccountValidationHandler handles AccountQuerySucceededEvent and performs business validation
func AccountValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		event, ok := e.(accountdomain.AccountQuerySucceededEvent)
		if !ok {
			return
		}

		logger.Info("validating account",
			"account_id", event.Query.AccountID,
			"user_id", event.Query.UserID,
		)

		// Perform business validation
		if event.Account == nil {
			logger.Error("account validation failed: account is nil")
			return
		}

		// Check if account has valid ID
		if event.Account.ID == uuid.Nil {
			logger.Error("account validation failed: account has invalid ID")
			_ = bus.Publish(ctx, accountdomain.AccountValidationFailedEvent{
				Query:   event.Query,
				Account: event.Account,
				Reason:  "account has invalid ID",
			})
			return
		}

		// Check if account belongs to the requesting user
		userID, err := uuid.Parse(event.Query.UserID)
		if err != nil {
			logger.Error("account validation failed: invalid user ID format")
			_ = bus.Publish(ctx, accountdomain.AccountValidationFailedEvent{
				Query:   event.Query,
				Account: event.Account,
				Reason:  "invalid user ID format",
			})
			return
		}

		if event.Account.UserID != userID {
			logger.Error("account validation failed: user not authorized for account")
			_ = bus.Publish(ctx, accountdomain.AccountValidationFailedEvent{
				Query:   event.Query,
				Account: event.Account,
				Reason:  "user not authorized for account",
			})
			return
		}

		// Validation passed
		logger.Info("account validation successful",
			"account_id", event.Query.AccountID,
		)

		// Convert AccountQuerySucceededEvent to AccountValidatedEvent
		_ = bus.Publish(ctx, accountdomain.AccountValidatedEvent(event))
	}
}
