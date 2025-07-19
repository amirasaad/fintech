package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
)

// WithdrawValidationHandler handles WithdrawRequestedEvent, performs validation, and publishes WithdrawValidatedEvent.
func WithdrawValidationHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawValidationHandler", "event_type", e.EventType())
		log.Info("🟢 [START] Received event", "event", e)
		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, we.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Account not found", "account_id", we.AccountID, "error", err)
			return
		}
		acc := mapper.MapAccountReadToDomain(accDto)
		if err := acc.ValidateWithdraw(we.UserID, we.Amount); err != nil {
			log.Error("❌ [ERROR] Account validation failed", "error", err)
			return
		}
		log.Info("✅ [SUCCESS] Account validated, emitting WithdrawValidatedEvent", "account_id", accDto.ID, "user_id", accDto.UserID)
		_ = bus.Publish(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: we,
			TargetCurrency:         accDto.Currency,
			Account:                acc,
		})
	}
}
