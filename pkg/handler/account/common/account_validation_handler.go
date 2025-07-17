package common

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/google/uuid"
)

// AccountValidationHandler handles AccountQuerySucceededEvent, maps DTO to domain, validates, and publishes AccountValidatedEvent.
func AccountValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		ev, ok := e.(events.AccountQuerySucceededEvent)
		if !ok {
			logger.Error("AccountValidationHandler: unexpected event type", "event", e)
			return
		}
		acc, err := MapDTOToAccount(ev.Result)
		if err != nil {
			logger.Error("AccountValidationHandler: failed to map DTO to domain Account", "error", err, "result", ev.Result)
			return
		}
		userID, err := uuid.Parse(ev.Result.UserID)
		if err != nil {
			logger.Error("AccountValidationHandler: invalid userID", "error", err, "userID", ev.Result.UserID)
			return
		}
		if err := acc.Validate(userID); err != nil {
			logger.Error("AccountValidationHandler: domain validation failed", "error", err)
			return
		}
		accountValidated := events.AccountValidatedEvent{
			AccountID: acc.ID.String(),
			UserID:    acc.UserID.String(),
			Amount:    acc.Balance.Amount(),
			Currency:  acc.Balance.Currency().String(),
		}
		_ = bus.Publish(ctx, accountValidated)
	}
}
