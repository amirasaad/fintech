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
)

// ValidationHandler validates the deposit request and emits DepositValidatedEvent on success.
func ValidationHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "RequestedHandler", "event_type", e.EventType())
		dr, ok := e.(events.DepositRequestedEvent)
		if !ok {
			log.Error("unexpected event type", "event", e)
			return
		}
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("failed to get AccountRepository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, dr.AccountID)
		if err != nil || accDto == nil {
			log.Error("account not found", "account_id", dr.AccountID, "error", err)
			return
		}
		acc := mapper.MapAccountReadToDomain(accDto)
		if err := acc.ValidateDeposit(dr.UserID, dr.Amount); err != nil {
			log.Error("account validation failed", "error", err)
			// TODO: Emit DepositValidationFailedEvent here per event-driven docs
			return
		}
		log.Info("account validated, emitting DepositValidatedEvent", "account_id", accDto.ID, "user_id", accDto.UserID)
		_ = bus.Publish(ctx, events.DepositValidatedEvent{
			DepositRequestedEvent: dr,
			AccountID:             acc.ID,
			Account:               acc,
		})
	}
}
