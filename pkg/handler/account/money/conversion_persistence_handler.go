package money

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/account"
	"github.com/amirasaad/fintech/pkg/repository"
)

// MoneyConversionPersistenceHandler handles MoneyConvertedEvent, persists the transaction with conversion info, and publishes a follow-up event if needed.
func MoneyConversionPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {
		slog.Info("MoneyConversionPersistenceHandler: received event", "event", e)
		mce, ok := e.(events.MoneyConvertedEvent)
		if !ok {
			slog.Error("MoneyConversionPersistenceHandler: unexpected event type", "event", e)
			return
		}
		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepo, err := uow.TransactionRepository()
			if err != nil {
				slog.Error("MoneyConversionPersistenceHandler: failed to get transaction repo", "error", err)
				return err
			}
			// Create a new transaction using the event data (generic for any transaction type)
			// You may want to use a factory or map event fields to your transaction model here
			// For now, we use deposit event as an example, but this should be generic
			// TODO: Replace with a generic transaction factory if needed
			newTx := account.NewDepositTransaction(mce.DepositRequestedEvent)
			moneyVal, err := money.NewMoneyFromSmallestUnit(mce.Amount, currency.Code(mce.Currency))
			if err != nil {
				slog.Error("MoneyConversionPersistenceHandler: failed to create Money value", "error", err)
				return err
			}
			newTx.Amount = moneyVal
			newTx.TargetCurrency = mce.TargetCurrency
			newTx.ConversionInfo = mce.ConversionInfo
			if err := txRepo.Update(newTx); err != nil {
				slog.Error("MoneyConversionPersistenceHandler: failed to create transaction", "error", err)
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
