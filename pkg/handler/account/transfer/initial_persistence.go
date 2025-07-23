package transfer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/mapper"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// InitialPersistence handles TransferValidatedEvent, creates an initial 'pending' transaction, and triggers conversion.
func InitialPersistence(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e domain.Event) error {
	return func(ctx context.Context, e domain.Event) error {
		log := logger.With("handler", "InitialPersistence", "event_type", e.Type())

		// 1. Defensive: Check event type and structure
		ve, ok := e.(events.TransferValidatedEvent)
		if !ok {
			log.Error("❌ [DISCARD] Unexpected event type", "event", e)
			return nil
		}
		log = log.With("correlation_id", ve.CorrelationID)
		log.Info("🟢 [START] Received event", "event", ve)

		if ve.AccountID == uuid.Nil || ve.UserID == uuid.Nil || ve.Amount.IsZero() || ve.Amount.IsNegative() {
			log.Error("❌ [DISCARD] Malformed validated event", "event", ve)
			return nil
		}

		// 2. Persist initial transaction (tx_out) atomically
		txID := ve.ID
		var destAccount *domain.Account
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
		conversionEvent := events.ConversionRequestedEvent{
			FlowEvent:     ve.FlowEvent,
			Amount:        ve.Amount,
			To:            destAccount.Currency(),
			RequestID:     txID.String(),
			Timestamp:     time.Now(),
			TransactionID: txID,
		}

		log.Info("📤 [EMIT] Emitting ConversionRequestedEvent", "event", conversionEvent)
		return bus.Emit(ctx, conversionEvent)
	}
}
