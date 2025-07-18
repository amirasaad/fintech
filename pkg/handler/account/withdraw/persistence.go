package withdraw

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// WithdrawPersistenceHandler handles WithdrawValidatedEvent: persists the withdraw transaction and emits WithdrawPersistedEvent.
func WithdrawPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		log := logger.With("handler", "WithdrawPersistenceHandler", "event_type", e.EventType())
		log.Info("received event", "event", e)

		ve, ok := e.(events.WithdrawValidatedEvent)
		if !ok {
			log.Error("unexpected event", "event", e)
			return
		}
		log.Info("received WithdrawValidatedEvent", "event", ve)

		txID := uuid.New()
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				log.Error("failed to get repo", "err", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return errors.New("failed to retrieve repo")
			}
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:        txID,
				UserID:    ve.UserID,
				AccountID: ve.AccountID,
				Amount:    ve.Amount.Amount(),
				Currency:  ve.Amount.Currency().String(),
				Status:    "created",
			}); err != nil {
				return err
			}
			log.Info("withdraw transaction persisted", "transaction_id", txID)
			return nil
		}); err != nil {
			log.Error("failed to persist withdraw transaction", "error", err)
			return
		}
		log.Info("emitting WithdrawPersistedEvent", "transaction_id", txID)
		_ = bus.Publish(ctx, events.WithdrawPersistedEvent{
			WithdrawValidatedEvent: ve,
			TransactionID:          txID,
		})

		// Emit CurrencyConversionRequested for the conversion handler chain
		_ = bus.Publish(ctx, events.CurrencyConversionRequested{
			EventID:        uuid.New(),
			TransactionID:  txID,
			AccountID:      ve.AccountID,
			UserID:         ve.UserID,
			Amount:         ve.Amount,
			SourceCurrency: ve.Amount.Currency().String(),
			TargetCurrency: ve.Amount.Currency().String(), // TODO: set actual target currency if different
			Timestamp:      ve.Timestamp,
		})
	}
}
