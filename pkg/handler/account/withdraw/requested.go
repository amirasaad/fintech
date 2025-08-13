package withdraw

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
)

// HandleRequested handles WithdrawRequested events by validating and persisting the withdraw.
// This follows the new event flow pattern: Requested -> HandleRequested (validate and persist).
func HandleRequested(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) func(
	ctx context.Context,
	e events.Event,
) error {
	return func(
		ctx context.Context,
		e events.Event,
	) error {
		log := logger.With(
			"handler", "withdraw.HandleRequested",
			"event_type", e.Type(),
		)
		log.Info("🟢 [START] Processing WithdrawRequested event")

		// Type assert to get the withdraw request
		wr, ok := e.(*events.WithdrawRequested)
		if !ok {
			log.Error(
				"❌ [ERROR] Unexpected event type",
				"expected", "WithdrawRequested",
				"got", e.Type(),
			)
			return fmt.Errorf("unexpected event type: %s", e.Type())
		}

		log = log.With(
			"user_id", wr.UserID,
			"account_id", wr.AccountID,
			"amount", wr.Amount.String(),
			"correlation_id", wr.CorrelationID,
		)

		// Validate the withdraw request
		if err := wr.Validate(); err != nil {
			log.Error(
				"❌ [ERROR] Withdraw validation failed",
				"error", err,
			)
			// Emit failed event
			failedEvent := events.NewWithdrawFailed(
				wr,
				err.Error(),
			)
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error(
					"❌ [ERROR] Failed to emit WithdrawFailed event",
					"error", err,
				)
			}
			return nil
		}

		// Create transaction ID
		txID := uuid.New()

		// Persist the withdraw transaction
		if err := persistWithdrawTransaction(ctx, uow, wr, txID, log); err != nil {
			log.Error(
				"❌ [ERROR] Failed to persist withdraw transaction",
				"error", err,
				"transaction_id", txID,
			)
			// Emit failed event
			failedEvent := events.NewWithdrawFailed(
				wr,
				fmt.Sprintf("failed to persist transaction: %v", err),
			)
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error(
					"❌ [ERROR] Failed to emit WithdrawFailed event",
					"error", err,
				)
			}
			return nil
		}

		log.Info("✅ [SUCCESS] Withdraw validated and persisted", "transaction_id", txID)

		// Emit CurrencyConversionRequested event
		ccr := events.NewCurrencyConversionRequested(
			wr.FlowEvent,
			wr,
			events.WithConversionAmount(wr.Amount),
			events.WithConversionTo(currency.Code("USD")),
		)

		if err := bus.Emit(ctx, ccr); err != nil {
			log.Error(
				"❌ [ERROR] Failed to emit CurrencyConversionRequested event",
				"error", err,
			)
			return fmt.Errorf(
				"failed to emit CurrencyConversionRequested event: %w", err,
			)
		}

		log.Info(
			"📤 [EMITTED] event",
			"event_id", ccr.ID,
			"event_type", ccr.Type(),
		)
		return nil
	}
}

// persistWithdrawTransaction persists the withdraw transaction to the database
func persistWithdrawTransaction(
	ctx context.Context,
	uow repository.UnitOfWork,
	wr *events.WithdrawRequested,
	txID uuid.UUID,
	log *slog.Logger,
) error {
	return uow.Do(ctx, func(uow repository.UnitOfWork) error {
		// Get the transaction repository
		txRepo, err := common.GetTransactionRepository(uow, log)
		if err != nil {
			return fmt.Errorf("failed to get transaction repository: %w", err)
		}

		// Create the transaction record using DTO
		txCreate := dto.TransactionCreate{
			ID:          txID,
			UserID:      wr.UserID,
			AccountID:   wr.AccountID,
			Amount:      wr.Amount.Amount(),
			Currency:    wr.Amount.Currency().String(),
			Status:      "created",
			MoneySource: "withdraw",
		}

		if err := txRepo.Create(ctx, txCreate); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		return nil
	})
}
