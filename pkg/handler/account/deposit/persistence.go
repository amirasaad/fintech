// Package deposit previously contained DepositPersistenceHandler, now moved to pkg/handler/payment/persistence_handler.go
package deposit

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
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

		// Convert float64 amount and currency to money.Money value object
		m, err := money.New(ve.Amount, currency.Code(ve.Currency))
		if err != nil {
			logger.Error("failed to convert amount to money.Money", "error", err)
			return
		}

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
			return txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txID,
				UserID:      ve.UserID,
				AccountID:   ve.AccountID,
				Amount:      m.Amount(),
				Currency:    m.Currency().String(),
				Status:      "created",
				MoneySource: ve.Source,
			})
		}); err != nil {
			logger.Error("failed to persist transaction", "error", err)
			return
		}

		// Emit DepositPersistedEvent for the next step in the event flow
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			MoneyCreatedEvent: events.MoneyCreatedEvent{
				DepositValidatedEvent: ve,
				Amount:                m.Amount(),
				Currency:              m.Currency().String(),
				TargetCurrency:        ve.Currency,
				TransactionID:         txID,
				UserID:                ve.UserID,
			},
			TransactionID: txID,
			UserID:        ve.UserID,
		})
	}
}
