// Package deposit previously contained DepositPersistenceHandler, now moved to pkg/handler/payment/persistence_handler.go
package deposit

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

// Persistence handles DepositValidatedEvent: converts the float64 amount and currency to money.Money, persists the transaction, and emits DepositPersistedEvent.
func Persistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "Persistence", "event_type", e.Type())
		log.Info("[DEPTH] Event received", "type", e.Type(), "event", e)
		log.Info("🟢 [START] Received event", "event", e)

		// Expect DepositValidatedEvent from validation handler
		ve, ok := e.(*events.DepositValidatedEvent)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event", "event", e)
			return nil
		}
		// Access correlation ID directly from the embedded FlowEvent
		correlationID := ve.CorrelationID
		if correlationID == uuid.Nil {
			correlationID = uuid.New()
		}
		log = log.With("correlation_id", correlationID)
		log.Info("🔄 [PROCESS] Received DepositValidatedEvent", "event", ve)

		// Log the currency of the validated deposit before persisting
		log.Info("[CHECK] DepositValidatedEvent amount currency before persist", "currency", ve.Amount.Currency().String())
		// ve.Amount should always be the source currency at this stage

		// Create a new transaction and persist it
		txID := uuid.New()
		var accountCurrency string

		if err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repoAny, err := uow.GetRepository((*account.Repository)(nil))
			if err != nil {
				log.Error("❌ [ERROR] Failed to get AccountRepository", "error", err)
				return err
			}
			repo := repoAny.(account.Repository)
			accDto, err := repo.Get(ctx, ve.AccountID)
			if err != nil {
				log.Error("❌ [ERROR] Failed to get Account", "account_id", ve.AccountID, "error", err)
				return err
			}
			if accDto == nil {
				log.Error("❌ [ERROR] Account not found", "account_id", ve.AccountID)
				return errors.New("account not found")
			}

			accountCurrency = accDto.Currency

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

			log.Info("✅ [SUCCESS] Transaction persisted", "transaction_id", txID, "correlation_id", correlationID)
			return nil
		}); err != nil {
			log.Error("❌ [ERROR] Failed to persist transaction", "error", err)
			return err
		}
		// Emit DepositPersistedEvent using factory function
		persistedEvent := events.NewDepositPersistedEvent(
			ve.UserID,
			ve.AccountID,
			ve.CorrelationID,
			events.WithDepositValidatedEventForPersisted(*ve),
			events.WithTransactionIDForPersisted(txID),
			events.WithDepositAmountForPersisted(ve.Amount),
		)

		log.Info("📤 [EMIT] Emitting DepositPersistedEvent", "event", persistedEvent, "correlation_id", correlationID.String())
		if err := bus.Emit(ctx, persistedEvent); err != nil {
			return err
		}

		// Only emit ConversionRequestedEvent if a conversion is needed and account currency is valid
		log.Info("📤 [EMIT] Emitting ConversionRequestedEvent for deposit", "transaction_id", txID, "correlation_id", correlationID)
		conversionEvent := events.NewConversionRequestedEvent(
			ve.FlowEvent,
			events.WithConversionAmount(ve.Amount),
			events.WithConversionTo(currency.Code(accountCurrency)),
			events.WithConversionRequestID(txID.String()),
			events.WithConversionTransactionID(txID),
			events.WithConversionTimestamp(time.Now()),
		)
		log.Info("📤 [EMIT] About to emit ConversionRequestedEvent", "handler", "Persistence", "event_type", conversionEvent.Type(), "correlation_id", correlationID.String())
		return bus.Emit(ctx, conversionEvent)
	}
}
