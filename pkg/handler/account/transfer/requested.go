package transfer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/repository"
)

// HandleRequested handles TransferValidatedEvent,
// creates an initial 'pending' transaction, and triggers conversion.
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
			"handler", "HandleRequested",
			"event_type", e.Type(),
		)

		tr, ok := e.(*events.TransferRequested)
		if !ok {
			log.Error(
				"❌ [DISCARD] Unexpected event type",
				"event", e,
			)
			return fmt.Errorf("unexpected event type: %T", e)
		}
		log = log.With("correlation_id", tr.CorrelationID)
		log.Info("🟢 [START] Received event", "event", tr)

		if err := tr.Validate(); err != nil {
			log.Error(
				"❌ [DISCARD] Malformed validated event",
				"error", err,
			)
			return err
		}

		// 2. Persist initial transaction (tx_out) atomically
		txID := tr.ID
		var destAccountRead *dto.AccountRead
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepo, err := common.GetTransactionRepository(uow, log)
			if err != nil {
				return fmt.Errorf("failed to get repo: %w", err)
			}

			accountRepo, err := common.GetAccountRepository(uow, log)
			if err != nil {
				return fmt.Errorf("failed to get account repo: %w", err)
			}
			destAccountRead, err = accountRepo.Get(ctx, tr.DestAccountID)
			if err != nil {
				return fmt.Errorf("failed to get destination account: %w", err)
			}
			return txRepo.Create(ctx, dto.TransactionCreate{
				ID:          txID,
				UserID:      tr.UserID,
				AccountID:   tr.AccountID,
				Amount:      tr.Amount.Negate().Amount(),
				Currency:    tr.Amount.Currency().String(),
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
		ccr := events.NewCurrencyConversionRequested(
			tr.FlowEvent,
			tr,
			events.WithConversionAmount(tr.Amount),
			events.WithConversionTo(money.Code(destAccountRead.Currency)),
			events.WithConversionTransactionID(txID),
		)

		log.Info(
			"📤 [EMIT] Emitting event",
			"event_type", ccr.Type(),
		)
		if err := bus.Emit(ctx, ccr); err != nil {
			log.Error(
				"❌ [ERROR] Failed to emit CurrencyConversionRequested",
				"error", err,
			)
		}
		return nil
	}
}
