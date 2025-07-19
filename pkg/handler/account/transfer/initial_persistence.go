package transfer

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// InitialPersistenceHandler handles TransferValidatedEvent: creates initial transaction record and triggers conversion.
func InitialPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork, logger *slog.Logger) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		logger := logger.With("handler", "TransferInitialPersistenceHandler")
		logger.Info("received event", "event", e)

		ve, ok := e.(events.TransferValidatedEvent)
		if !ok {
			logger.Error("unexpected event type", "event", e)
			return
		}
		logger.Info("received TransferValidatedEvent", "event", ve)

		txID := uuid.New()
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				logger.Error("failed to get repo", "err", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				return err
			}
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txID,
				UserID:      ve.SenderUserID,
				AccountID:   ve.SourceAccountID,
				Amount:      -ve.Amount.Amount(), // Negative amount for outgoing transaction
				Currency:    ve.Amount.Currency().String(),
				Status:      "created",
				MoneySource: "transfer",
			}); err != nil {
				return err
			}
			logger.Info("outgoing transfer transaction created", "transaction_id", txID, "amount", -ve.Amount.Amount())
			return nil
		}); err != nil {
			logger.Error("failed to create transfer transaction", "error", err)
			return
		}

		// For transfer, we need to determine the target currency from the destination account
		// For now, we'll use the same currency as the source (no conversion needed)
		targetCurrency := ve.Amount.Currency().String()

		logger.Info("emitting ConversionRequested for transfer", "transaction_id", txID)
		_ = bus.Publish(ctx, events.ConversionRequested{
			CorrelationID:  txID.String(),
			FlowType:       "transfer",
			OriginalEvent:  ve,
			Amount:         ve.Amount,
			SourceCurrency: ve.Amount.Currency().String(),
			TargetCurrency: targetCurrency,
		})
	}
}
