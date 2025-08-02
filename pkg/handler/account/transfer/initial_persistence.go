package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/mapper"

	domainaccount "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
)

// InitialPersistence handles TransferValidatedEvent, creates an initial 'pending' transaction, and triggers conversion.
func InitialPersistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e events.Event) error {
	return func(ctx context.Context, e events.Event) error {
		log := logger.With("handler", "InitialPersistence", "event_type", e.Type())

		ve, ok := e.(*events.TransferRequested)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return fmt.Errorf("unexpected event type: %T", e)
		}
		log = log.With("correlation_id", ve.CorrelationID)
		log.Info("🟢 [START] Received event", "event", ve)

		if err := ve.Validate(); err != nil {
			log.Error("❌ [DISCARD] Malformed validated event", "error", err)
			return err
		}

		// 2. Persist initial transaction (tx_out) atomically
		txID := ve.ID
		var destAccount *domainaccount.Account
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			repoAny, err := uow.GetRepository((*transaction.Repository)(nil))
			if err != nil {
				return fmt.Errorf("failed to get repo: %w", err)
			}

			txRepo, ok := repoAny.(transaction.Repository)
			if !ok {
				return fmt.Errorf("unexpected repo type")
			}
			accountRepoAny, err := uow.GetRepository((*account.Repository)(nil))
			if err != nil {
				return fmt.Errorf("failed to get account repo: %w", err)
			}
			accountRepo, ok := accountRepoAny.(account.Repository)
			if !ok {
				return fmt.Errorf("unexpected account repo type")
			}
			destAccountRead, err := accountRepo.Get(ctx, ve.DestAccountID)
			if err != nil {
				return fmt.Errorf("failed to get destination account: %w", err)
			}
			destAccount = mapper.MapAccountReadToDomain(destAccountRead)
			return txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txID,
				UserID:      ve.UserID,
				AccountID:   ve.AccountID,
				Amount:      ve.Amount.Negate().Amount(),
				Currency:    ve.Amount.Currency().String(),
				Status:      "pending",
				MoneySource: "transfer",
			})
		})

		if err != nil {
			log.Error("❌ [ERROR] Failed to create initial transaction", "error", err)
			return err
		}
		log.Info("✅ [SUCCESS] Initial 'pending' transaction created", "transaction_id", txID)

		// 3. Emit event to trigger currency conversion
		conversionEvent := events.NewCurrencyConversionRequested(
			ve.FlowEvent,
			events.WithConversionAmount(ve.Amount),
			events.WithConversionTo(destAccount.Currency()),
			events.WithConversionTransactionID(txID),
		)

		log.Info("📤 [EMIT] Emitting ConversionRequested", "event", conversionEvent)
		if err := bus.Emit(ctx, conversionEvent); err != nil {
			log.Error("❌ [ERROR] Failed to emit ConversionRequested", "error", err)
			// If we fail to emit the conversion event, emit a failed event
			err = bus.Emit(ctx, events.NewTransferFailed(
				ve.FlowEvent,
				"failed to emit conversion event: "+err.Error(),
			))
			if err != nil {
				log.Error("❌ [CRITICAL] Failed to emit TransferFailedEvent", "error", err)
			}
			return fmt.Errorf("failed to emit conversion event: %w", err)
		}
		return nil
	}
}
