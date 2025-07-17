package account

import (
	"context"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
)

// WithdrawValidationHandler handles WithdrawRequestedEvent, performs validation, and publishes WithdrawValidatedEvent.
func WithdrawValidationHandler(bus eventbus.EventBus, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			logger.Error("WithdrawValidationHandler: unexpected event type", "event", e)
			return
		}
		if we.UserID == "" || we.AccountID == "" || we.Amount <= 0 || we.Currency == "" {
			logger.Error("WithdrawValidationHandler: missing or invalid fields", "event", we)
			return
		}
		userUUID, err := uuid.Parse(we.UserID)
		if err != nil {
			logger.Error("WithdrawValidationHandler: invalid userID", "error", err, "userID", we.UserID)
			return
		}
		getAccountResult := queries.GetAccountResult{
			AccountID: we.AccountID,
			UserID:    we.UserID,
			Balance:   we.Amount,
			Currency:  we.Currency,
		}
		acc, err := MapDTOToAccount(getAccountResult)
		if err != nil {
			logger.Error("WithdrawValidationHandler: failed to map DTO to domain Account", "error", err, "result", getAccountResult)
			return
		}
		if err := acc.Validate(userUUID); err != nil {
			logger.Error("WithdrawValidationHandler: domain validation failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.WithdrawValidatedEvent{WithdrawRequestedEvent: we})
	}
}
