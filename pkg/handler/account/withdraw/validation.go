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
	"github.com/google/uuid"
)

// WithdrawValidationHandler handles WithdrawRequestedEvent, performs validation, and publishes WithdrawValidatedEvent.
func WithdrawValidationHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "WithdrawValidationHandler", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)
		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return nil
		}
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
			return nil
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, we.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Account not found", "account_id", we.AccountID, "error", err)
			return nil
		}
		acc := mapper.MapAccountReadToDomain(accDto)
		if err := acc.ValidateWithdraw(we.UserID, we.Amount); err != nil {
			log.Error("❌ [ERROR] Account validation failed", "error", err)
			return nil
		}
		correlationID := uuid.New()
		validatedEvent := events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: we,
			TargetCurrency:         accDto.Currency,
			Account:                acc,
		}
		log.Info("✅ [SUCCESS] Withdraw validated, emitting WithdrawValidatedEvent", "account_id", accDto.ID, "user_id", accDto.UserID, "correlation_id", correlationID.String())
		return bus.Emit(ctx, validatedEvent)
	}
}
