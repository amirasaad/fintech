package deposit

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/mapper"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// ValidationHandler validates the deposit request and emits DepositValidatedEvent on success.
func ValidationHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "DepositValidationHandler", "event_type", e.Type())
		dr, ok := e.(events.DepositRequestedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "event", e)
			return
		}
		correlationID := uuid.New()
		log = log.With("correlation_id", correlationID)
		log.Info("🟢 [START] Received event", "event", e)
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, dr.AccountID)
		if err != nil || accDto == nil {
			log.Error("❌ [ERROR] Account not found", "account_id", dr.AccountID, "error", err)
			return
		}
		acc := mapper.MapAccountReadToDomain(accDto)
		if err := acc.ValidateDeposit(dr.UserID, dr.Amount); err != nil {
			log.Error("❌ [ERROR] Account validation failed", "error", err)
			return
		}
		validatedEvent := events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
			Account:               acc,
		}
		log.Info("✅ [SUCCESS] Account validated, emitting DepositValidatedEvent", "account_id", accDto.ID, "user_id", accDto.UserID, "correlation_id", correlationID.String())
		_ = bus.Publish(ctx, validatedEvent)
	}
}
