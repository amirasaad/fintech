// Package app provides functionality for setting up and configuring the event Bus
// with all necessary event handlers for the application.
package app

import (
	"github.com/amirasaad/fintech/pkg/currency"
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/amirasaad/fintech/pkg/handler/conversion"
	"github.com/amirasaad/fintech/pkg/handler/fees"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/repository"
)

// setupEventBus registers all event handlers with the provided event Bus.
func (a *App) setupEventBus() {

	bus := a.Deps.EventBus
	uow := a.Deps.Uow
	currencyConverter := a.Deps.CurrencyConverter
	logger := a.Deps.Logger

	a.setupConversionHandlers(bus, uow, currencyConverter, logger)

	a.setupDepositHandlers(bus, uow, logger)
	a.setupWithdrawHandlers(bus, uow, logger)
	a.setupPaymentHandlers(bus, uow, logger)
	a.setupTransferHandlers(bus, uow, logger)
	a.setupFeesHandlers(bus, uow, logger)
}

func (a *App) setupWithdrawHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	bus.Register(
		events.EventTypeWithdrawRequested,
		withdraw.HandleRequested(
			bus,
			uow,
			logger,
		),
	)
	bus.Register(
		events.EventTypeWithdrawCurrencyConverted,
		withdraw.HandleCurrencyConverted(
			bus,
			uow,
			logger,
		),
	)
}

func (a *App) setupPaymentHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	bus.Register(
		events.EventTypePaymentInitiated,
		payment.HandleInitiated(
			bus,
			a.Deps.PaymentProvider,
			logger,
		),
	)
	bus.Register(
		events.EventTypePaymentProcessed,
		payment.HandleProcessed(
			uow,
			logger,
		),
	)
	bus.Register(
		events.EventTypePaymentCompleted,
		payment.HandleCompleted(
			bus,
			uow,
			logger,
		),
	)

}

func (a *App) setupFeesHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	bus.Register(
		events.EventTypeFeesCalculated,
		fees.HandleCalculated(
			uow,
			logger,
		),
	)
}

func (a *App) setupTransferHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	bus.Register(
		events.EventTypeTransferRequested,
		transfer.HandleRequested(
			bus,
			uow,
			logger,
		),
	)
	bus.Register(
		events.EventTypeTransferCurrencyConverted,
		transfer.HandleCurrencyConverted(
			bus,
			uow,
			logger,
		),
	)
	bus.Register(
		events.EventTypeTransferCompleted,
		transfer.HandleCompleted(
			bus,
			uow,
			logger,
		),
	)
}

func (a *App) setupDepositHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	bus.Register(
		events.EventTypeDepositRequested,
		deposit.HandleRequested(
			bus,
			uow,
			logger,
		),
	)
	bus.Register(
		events.EventTypeDepositCurrencyConverted,
		deposit.HandleCurrencyConverted(
			bus,
			uow,
			logger,
		),
	)
}

func (a *App) setupConversionHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	currencyConverter currency.Converter,
	logger *slog.Logger,
) {
	// 1️⃣ GENERIC CONVERSION HANDLER
	// Handles all ConversionRequestedEvent by delegating to a flow-specific factory.
	conversionFactories := map[string]conversion.EventFactory{
		"deposit":  &conversion.DepositEventFactory{},
		"withdraw": &conversion.WithdrawEventFactory{},
		"transfer": &conversion.TransferEventFactory{},
	}
	bus.Register(
		events.EventTypeCurrencyConversionRequested,
		conversion.HandleRequested(
			bus,
			currencyConverter,
			logger,
			conversionFactories,
		),
	)
	bus.Register(
		events.EventTypeCurrencyConverted,
		conversion.HandleCurrencyConverted(
			uow,
			logger,
		),
	)
}
