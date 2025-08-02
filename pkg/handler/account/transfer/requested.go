package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// RequestedHandler handles TransferRequested events by validating and persisting the transfer.
// This follows the new event flow pattern: Requested -> RequestedHandler (validate and persist).
func RequestedHandler(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "TransferRequestedHandler", "event_type", e.Type())
		log.Info("🟢 [START] Processing TransferRequested event")

		// Type assert to get the transfer request
		tr, ok := e.(*events.TransferRequested)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "expected", "TransferRequested", "got", e.Type())
			return fmt.Errorf("unexpected event type: %s", e.Type())
		}

		log = log.With(
			"user_id", tr.UserID,
			"account_id", tr.AccountID,
			"dest_account_id", tr.DestAccountID,
			"amount", tr.Amount.String(),
			"correlation_id", tr.CorrelationID,
		)

		// Validate the transfer request
		if err := tr.Validate(); err != nil {
			log.Error("❌ [ERROR] Transfer validation failed", "error", err)
			// Emit failed event
			failedEvent := events.NewTransferFailed(
				tr.FlowEvent,
				err.Error(),
			)
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error("❌ [ERROR] Failed to emit TransferFailed event", "error", err)
			}
			return nil
		}

		// Create transaction ID
		txID := uuid.New()

		// Persist the transfer transaction
		if err := persistTransferTransaction(ctx, uow, tr, txID); err != nil {
			log.Error("❌ [ERROR] Failed to persist transfer transaction", "error", err, "transaction_id", txID)
			// Emit failed event
			failedEvent := events.NewTransferFailed(
				tr.FlowEvent,
				fmt.Sprintf("failed to persist transaction: %v", err),
			)
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error("❌ [ERROR] Failed to emit TransferFailed event", "error", err)
			}
			return nil
		}

		log.Info("✅ [SUCCESS] Transfer validated and persisted", "transaction_id", txID)

		// Emit CurrencyConversionRequested event
		ccr := events.NewCurrencyConversionRequested(
			tr.FlowEvent,
			events.WithConversionAmount(tr.Amount),
			events.WithConversionTo(currency.Code("USD")), // This should come from account currency
			events.WithConversionTransactionID(txID),
		)

		if err := bus.Emit(ctx, ccr); err != nil {
			log.Error("❌ [ERROR] Failed to emit CurrencyConversionRequested event", "error", err)
			return fmt.Errorf("failed to emit CurrencyConversionRequested event: %w", err)
		}

		log.Info("📤 [PUBLISHED] CurrencyConversionRequested event", "event_id", ccr.ID)
		return nil
	}
}

// persistTransferTransaction persists the transfer transaction to the database
func persistTransferTransaction(ctx context.Context, uow repository.UnitOfWork, tr *events.TransferRequested, txID uuid.UUID) error {
	return uow.Do(ctx, func(uow repository.UnitOfWork) error {
		// Get the transaction repository
		txRepoAny, err := uow.GetRepository((*transaction.Repository)(nil))
		if err != nil {
			return fmt.Errorf("failed to get transaction repository: %w", err)
		}
		txRepo, ok := txRepoAny.(transaction.Repository)
		if !ok {
			return fmt.Errorf("failed to get transaction repository: %w", err)
		}

		// Create the transaction record using DTO
		txCreate := dto.TransactionCreate{
			ID:          txID,
			UserID:      tr.UserID,
			AccountID:   tr.AccountID,
			Amount:      tr.Amount.Amount(),
			Currency:    tr.Amount.Currency().String(),
			Status:      "created",
			MoneySource: tr.Source,
		}

		if err := txRepo.Create(ctx, txCreate); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		return nil
	})
}
