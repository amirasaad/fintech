// Package deposit previously contained DepositPersistenceHandler, now moved to pkg/handler/payment/persistence_handler.go
package deposit

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// PersistenceHandler handles DepositValidatedEvent: converts the float64 amount and currency to money.Money, persists the transaction, and emits DepositPersistedEvent.
func PersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With(
			"handler", "PersistenceHandler",
			"event_type", e.EventType(),
		)
		logger.Info("received event", "event", e)

		// Expect DepositValidatedEvent from validation handler
		ve, ok := e.(events.DepositValidatedEvent)
		if !ok {
			logger.Error("unexpected event", "event", e)
			return
		}
		logger.Info("received DepositValidatedEvent", "event", ve)

		// Create a new transaction and persist it
		txID := uuid.New()
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("failed to get repo", "err", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return errors.New("failed to retrieve repo")
			}
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txID,
				UserID:      ve.UserID,
				AccountID:   ve.AccountID,
				Amount:      ve.Amount.Amount(),
				Currency:    ve.Amount.Currency().String(),
				Status:      "created",
				MoneySource: ve.Source,
			}); err != nil {
				return err
			}
			logger.Info("transaction persisted", "transaction_id", txID)
			return nil
		}); err != nil {
			logger.Error("failed to persist transaction", "error", err)
			return
		}
		logger.Info("emitting CurrencyConversionRequested", "transaction_id", txID)

		// Emit CurrencyConversionRequested for the new conversion handler chain
		_ = bus.Publish(ctx, events.CurrencyConversionRequested{
			EventID:        uuid.New(),
			UserID:         ve.UserID,
			AccountID:      ve.AccountID,
			TransactionID:  txID,
			Amount:         ve.Amount,
			SourceCurrency: ve.Amount.Currency().String(),
			TargetCurrency: ve.Account.Balance.Currency().String(), // Use ve.Currency as default; update if target differs
			Timestamp:      ve.Timestamp,
		})
	}
}
