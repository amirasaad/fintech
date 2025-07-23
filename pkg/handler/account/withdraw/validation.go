package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// Validation handles WithdrawRequestedEvent, performs initial stateless validation, and publishes WithdrawValidatedEvent.
func Validation(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "Validation", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}

		if we.AccountID == uuid.Nil || !we.Amount.IsPositive() {
			log.Error("❌ [ERROR] Invalid withdrawal request", "event", we)
			if err := bus.Emit(ctx, events.WithdrawFailedEvent{WithdrawRequestedEvent: we, Reason: "Invalid withdrawal request data"}); err != nil {
				log.Error("failed to emit WithdrawFailedEvent", "error", err)
			}
			return nil
		}

		accRepoAny, err := uow.GetRepository((*account.Repository)(nil))

		if err != nil {
			return err
		}
		accRepo, ok := accRepoAny.(account.Repository)
		if !ok {
			return errors.New("failed to get repo")
		}

		accDto, err := accRepo.Get(ctx, we.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err)
			return bus.Emit(ctx, events.WithdrawFailedEvent{WithdrawRequestedEvent: we, Reason: "Account not found"})
		}

		acc := mapper.MapAccountReadToDomain(accDto)

		validatedEvent := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: we,
			Account:                acc,
			TargetCurrency:         accDto.Currency,
		}

		log.Info("✅ [SUCCESS] Withdraw request validated, emitting WithdrawValidatedEvent", "account_id", we.AccountID, "user_id", we.UserID)
		return bus.Emit(ctx, validatedEvent)
	}
}
