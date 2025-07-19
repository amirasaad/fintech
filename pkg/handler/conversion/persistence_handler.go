// Package conversion handles currency conversion events and persistence logic.
package conversion

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
)

// PersistenceHandler persists CurrencyConversionDone events and emits CurrencyConversionPersisted.
func PersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With(
			"handler", "PersistenceHandler",
			"event_type", e.EventType(),
		)
		logger.Info("received event", "event", e)

		ce, ok := e.(events.CurrencyConversionDone)
		if !ok {
			logger.Error("unexpected event", "event", e)
			return
		}

		// Persist conversion result (stubbed for now)
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			//TODO:
			return nil
		}); err != nil {
			logger.Error("failed to persist conversion data", "error", err)
			return
		}

		logger.Info("conversion persisted", "transaction_id", ce.TransactionID)
	}
}
