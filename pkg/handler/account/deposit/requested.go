package deposit

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
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
)

// Requested handles DepositRequested events by validating and persisting the deposit.
// This follows the new event flow pattern: Requested -> Requested (validate and persist).
func Requested(bus eventbus.Bus, uow repository.UnitOfWork, logger *slog.Logger) func(ctx context.Context, e common.Event) error {
	return func(ctx context.Context, e common.Event) error {
		log := logger.With("handler", "DepositRequestedHandler", "event_type", e.Type())
		log.Info("🟢 [START] Processing DepositRequested event")

		// Type assert to get the deposit request
		dr, ok := e.(*events.DepositRequested)
		if !ok {
			log.Error("❌ [ERROR] Unexpected event type", "expected", "DepositRequested", "got", e.Type())
			return fmt.Errorf("unexpected event type: %s", e.Type())
		}

		log = log.With(
			"user_id", dr.UserID,
			"account_id", dr.AccountID,
			"amount", dr.Amount.String(),
			"correlation_id", dr.CorrelationID,
		)

		// Validate the deposit request
		if err := dr.Validate(); err != nil {
			log.Error("❌ [ERROR] Deposit validation failed", "error", err)
			// Emit failed event
			failedEvent := events.NewDepositFailed(*dr, err.Error())
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error("❌ [ERROR] Failed to emit DepositFailed event", "error", err)
			}
			return nil
		}

		accountRepoAny, err := uow.GetRepository((*repository.AccountRepository)(nil))
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return fmt.Errorf("failed to get account repository: %w", err)
		}
		accountRepo, ok := accountRepoAny.(account.Repository)
		if !ok {
			log.Error("❌ [ERROR] Failed to get account repository", "error", err)
			return fmt.Errorf("failed to get account repository: %w", err)
		}

		account, err := accountRepo.Get(ctx, dr.AccountID)
		if err != nil {
			log.Error("❌ [ERROR] Failed to get account", "error", err)
			return fmt.Errorf("failed to get account: %w", err)
		}

		// Create transaction ID if not provided
		if dr.TransactionID == uuid.Nil {
			dr.TransactionID = uuid.New()
		}

		// Persist the deposit transaction
		txID := dr.TransactionID
		if err := persistDepositTransaction(ctx, uow, dr); err != nil {
			log.Error("❌ [ERROR] Failed to persist deposit transaction", "error", err, "transaction_id", txID)
			// Emit failed event
			failedEvent := events.NewDepositFailed(*dr, fmt.Sprintf("failed to persist transaction: %v", err))
			if err := bus.Emit(ctx, failedEvent); err != nil {
				log.Error("❌ [ERROR] Failed to emit DepositFailed event", "error", err)
			}
			return nil
		}

		log.Info("✅ [SUCCESS] Deposit validated and persisted", "transaction_id", txID)

		// Emit CurrencyConversionRequested event
		conversionEvent := events.NewCurrencyConversionRequested(
			dr.FlowEvent,
			events.WithConversionAmount(dr.Amount),
			events.WithConversionTo(currency.Code(account.Currency)),
		)
		if err := bus.Emit(ctx, conversionEvent); err != nil {
			log.Error("❌ [ERROR] Failed to emit CurrencyConversionRequested event", "error", err)
			return fmt.Errorf("failed to emit CurrencyConversionRequested event: %w", err)
		}

		log.Info("📤 [PUBLISHED] CurrencyConversionRequested event", "event_id", conversionEvent.ID)
		return nil
	}
}

// persistDepositTransaction persists the deposit transaction to the database
func persistDepositTransaction(ctx context.Context, uow repository.UnitOfWork, dr *events.DepositRequested) error {
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

		// Create the transaction record using domain object
		tx := dto.TransactionCreate{
			ID:          dr.TransactionID,
			UserID:      dr.UserID,
			AccountID:   dr.AccountID,
			Amount:      dr.Amount.Amount(),
			Status:      "created",
			MoneySource: "deposit",
			Currency:    dr.Amount.Currency().String(),
		}

		if err := txRepo.Create(ctx, tx); err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}

		return nil
	})
}
