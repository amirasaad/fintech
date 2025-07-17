package payment

import (
	"context"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account/events"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	accounttx "github.com/amirasaad/fintech/pkg/handler/account"
	"github.com/amirasaad/fintech/pkg/repository"
)

// DepositPersistenceHandler handles MoneyConvertedEvent, persists to DB, and publishes DepositPersistedEvent.
func DepositPersistenceHandler(bus eventbus.EventBus, uow repository.UnitOfWork) func(context.Context, domain.Event) {
	return func(ctx context.Context, e domain.Event) {

		slog.Info("DepositPersistenceHandler: received event", "event", e)

		var (
			conversionInfo *common.ConversionInfo
			amount         int64
			curr           string
			depositEvent   events.DepositRequestedEvent
		)

		switch evt := e.(type) {
		case events.MoneyConvertedEvent:
			conversionInfo = evt.ConversionInfo
			amount = evt.Amount
			curr = evt.Currency
			depositEvent = evt.DepositRequestedEvent
		case events.MoneyCreatedEvent:
			amount = evt.Amount
			curr = evt.Currency
			depositEvent = evt.DepositRequestedEvent
		default:
			slog.Error("DepositPersistenceHandler: unexpected event type", "event", e)
			return
		}

		err := uow.Do(ctx, func(uow repository.UnitOfWork) error {
			txRepo, err := uow.TransactionRepository()
			if err != nil {
				slog.Error("DepositPersistenceHandler: failed to get transaction repo", "error", err)
				return err
			}
			newTx := accounttx.NewDepositTransaction(depositEvent)
			moneyVal, err := money.NewMoneyFromSmallestUnit(amount, currency.Code(curr))
			if err != nil {
				slog.Error("DepositPersistenceHandler: failed to create Money value", "error", err)
				return err
			}
			newTx.Amount = moneyVal
			// Set TargetCurrency and ConversionInfo
			newTx.TargetCurrency = curr
			newTx.ConversionInfo = conversionInfo
			if err := txRepo.Create(newTx, conversionInfo, depositEvent.Source); err != nil {
				slog.Error("DepositPersistenceHandler: failed to create transaction", "error", err)
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("DepositPersistenceHandler: persistence failed", "error", err)
			return
		}
		_ = bus.Publish(ctx, events.DepositPersistedEvent{
			MoneyCreatedEvent: events.MoneyCreatedEvent{
				DepositValidatedEvent: events.DepositValidatedEvent{
					DepositRequestedEvent: depositEvent,
					AccountID:             depositEvent.AccountID,
				},
				Amount:         amount,
				Currency:       curr,
				TargetCurrency: curr,
			},
			// Add DB transaction info if needed
		})
	}
}
