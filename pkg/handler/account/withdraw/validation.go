package withdraw

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/mapper"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
)

// WithdrawValidationHandler handles WithdrawRequestedEvent, performs validation, and publishes WithdrawValidatedEvent.
func WithdrawValidationHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawValidationHandler", "event_type", e.EventType())
		we, ok := e.(events.WithdrawRequestedEvent)
		if !ok {
			log.Error("unexpected event type", "event", e)
			return
		}
		if we.Amount.AmountFloat() <= 0 {
			log.Error("amount must be positive", "event", we)
			return
		}
		if we.UserID == uuid.Nil || we.AccountID == uuid.Nil {
			log.Error("missing or invalid fields", "event", we)
			return
		}
		repoAny, err := uow.GetRepository((*account.Repository)(nil))
		if err != nil {
			log.Error("failed to get AccountRepository", "error", err)
			return
		}
		repo := repoAny.(account.Repository)
		accDto, err := repo.Get(ctx, we.AccountID)
		if err != nil || accDto == nil {
			log.Error("account not found", "account_id", we.AccountID, "error", err)
			return
		}
		acc := mapper.MapAccountReadToDomain(accDto)
		if err := acc.Withdraw(we.UserID, we.Amount, accountdomain.ExternalTarget{}, ""); err != nil {
			log.Error("account validation failed", "error", err)
			return
		}
		log.Info("account validated, emitting WithdrawValidatedEvent", "account_id", accDto.ID, "user_id", accDto.UserID)
		_ = bus.Publish(ctx, events.WithdrawValidatedEvent{
			WithdrawRequestedEvent: we,
		})
	}
}
