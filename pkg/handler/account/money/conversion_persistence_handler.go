package money

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
)

// MoneyConversionPersistenceHandler handles MoneyConvertedEvent, updates the transaction with conversion info using CQRS repository, and publishes a follow-up event if needed.
func MoneyConversionPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		slog.Info("MoneyConversionPersistenceHandler: received event", "event", e)
		mce, ok := e.(events.MoneyConvertedEvent)
		if !ok {
			slog.Error("MoneyConversionPersistenceHandler: unexpected event type", "event", e)
			return
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				slog.Error("MoneyConversionPersistenceHandler: failed to get transaction repo", "error", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				slog.Error("MoneyConversionPersistenceHandler: failed to get transaction repo")
				return errors.New("MoneyConversionPersistenceHandler: failed to retrieve repo type")
			}

			// Prepare update DTO
			update := dto.TransactionUpdate{
				// Add conversion info fields as needed (extend DTO if required)
				// For now, no direct fields in DTO, so this is a placeholder

			}
			if err := txRepo.Update(ctx, mce.TransactionID, update); err != nil {
				slog.Error("MoneyConversionPersistenceHandler: failed to update transaction", "error", err)
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("MoneyConversionPersistenceHandler: persistence failed", "error", err)
			return
		}
		// Optionally publish a follow-up event here
	}
}
