package withdraw

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// Persistence handles WithdrawValidatedEvent: persists the withdraw transaction and emits WithdrawPersistedEvent.
func Persistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "Persistence", "event_type", e.Type())
		log.Info("🟢 [START] Received event", "event", e)

		ve, ok := e.(*events.WithdrawValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event", "event", e)
			return nil
		}
		correlationID := ve.CorrelationID
		if correlationID == uuid.Nil {
			correlationID = uuid.New()
		}
		log = log.With("correlation_id", correlationID)
		log.Info("🔄 [PROCESS] Received WithdrawValidatedEvent", "event", ve)

		txID := uuid.New()
		var accDto *dto.AccountRead
		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repoAny, err := uow.GetRepository((*account.Repository)(nil))
			if err != nil {
				log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
				return nil
			}
			repo := repoAny.(account.Repository)
			accDto, err = repo.Get(ctx, ve.AccountID)
			if err != nil || accDto == nil {
				log.Error("❌ [ERROR] Account not found", "account_id", ve.AccountID, "error", err)
				return nil
			}

			txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				log.Error("❌ [ERROR] Failed to get repo", "err", err)
				return err
			}
			txRepo, ok := txRepoAny.(transaction.Repository)
			if !ok {
				log.Error("❌ [ERROR] Failed to retrieve repo type")
				return errors.New("failed to retrieve repo type")
			}
			log.Debug("[DEBUG] About to persist transaction", "amount", ve.Amount.Amount(), "currency", ve.Amount.Currency().String())
			if err := txRepo.Create(ctx, dto.TransactionCreate{
				ID:                   txID,
				UserID:               ve.UserID,
				AccountID:            ve.AccountID,
				Amount:               ve.Amount.Amount(),
				Currency:             ve.Amount.Currency().String(),
				Status:               "created",
				ExternalTargetMasked: "", // TODO: Persist masked bank account number
			}); err != nil {
				return err
			}
			log.Info("✅ [SUCCESS] Withdraw transaction persisted", "transaction_id", txID)
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist withdraw transaction", "error", err)
			return nil
		}
		persistedEvent := events.NewWithdrawPersistedEvent(
			ve.FlowEvent,
			*ve,
			events.WithWithdrawTransactionID(txID),
		)
		log.Info("📤 [EMIT] Emitting WithdrawPersistedEvent", "event", persistedEvent, "correlation_id", correlationID.String())
		if err := bus.Emit(ctx, persistedEvent); err != nil {
			return err
		}

		// Emit ConversionRequested to trigger currency conversion for withdraw (decoupled from payment)
		log.Info("DEBUG: ve.UserID and ve.AccountID", "user_id", ve.UserID, "account_id", ve.AccountID)
		log.Info("DEBUG: Account Currency", "currency", currency.Code(accDto.Currency))

		conversionEvent := events.NewConversionRequestedEvent(
			ve.FlowEvent,
			events.WithConversionAmount(ve.Amount),
			events.WithConversionTo(currency.Code(accDto.Currency)),
			events.WithConversionRequestID(txID.String()),
			events.WithConversionTransactionID(txID),
			events.WithConversionTimestamp(time.Now()),
		)
		log.Info("DEBUG: Full ConversionRequestedEvent", "event", conversionEvent)
		log.Info("📤 [EMIT] About to emit ConversionRequestedEvent", "handler", "Persistence", "event_type", conversionEvent.Type(), "correlation_id", correlationID.String())
		return bus.Emit(ctx, conversionEvent)
	}
}
