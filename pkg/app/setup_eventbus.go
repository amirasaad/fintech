// Package app provides functionality for setting up and configuring the event Bus
// with all necessary event handlers for the application.
package app

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	handlercommon "github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/handler/conversion"
	"github.com/amirasaad/fintech/pkg/handler/fees"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/provider/exchange"

	"github.com/amirasaad/fintech/pkg/repository"
)

// setupEventBus registers all event handlers with the provided event Bus.
func (a *App) setupEventBus() {

	bus := a.Deps.EventBus
	uow := a.Deps.Uow
	logger := a.Deps.Logger

	a.setupConversionHandlers(
		bus,
		uow,
		a.Deps.ExchangeRateProvider,
		logger,
	)

	a.setupDepositHandlers(bus, uow, logger)
	a.setupWithdrawHandlers(bus, uow, logger)
	a.setupPaymentHandlers(bus, uow, logger)
	a.setupTransferHandlers(bus, uow, logger)
	a.setupFeesHandlers(bus, uow, logger)
	a.setupUserHandlers(bus, uow, logger)

}

func (a *App) setupUserHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	// bus.Register(
	// 	events.EventTypeUserOnboardingCompleted,
	// 	user.HandleUserOnboardingCompleted(
	// 		bus,
	// 		uow,
	// 		logger,
	// 	),
	//)
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
	bus.Register(
		events.EventTypeWithdrawValidated,
		withdraw.HandleValidated(
			bus,
			uow,
			a.Deps.PaymentProvider,
			a.Deps.Logger,
		),
	)
}

func (a *App) setupPaymentHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	logger *slog.Logger,
) {
	// Create idempotency trackers for each handler
	initiatedTracker := handlercommon.NewIdempotencyTracker()
	processedTracker := handlercommon.NewIdempotencyTracker()
	completedTracker := handlercommon.NewIdempotencyTracker()

	// Register handlers with idempotency middleware
	bus.Register(
		events.EventTypePaymentInitiated,
		handlercommon.WithIdempotency(
			payment.HandleInitiated(
				bus,
				a.Deps.PaymentProvider,
				logger,
			),
			initiatedTracker,
			payment.ExtractPaymentInitiatedKey,
			"HandleInitiated",
			logger,
		),
	)
	bus.Register(
		events.EventTypePaymentProcessed,
		handlercommon.WithIdempotency(
			payment.HandleProcessed(
				uow,
				logger,
			),
			processedTracker,
			payment.ExtractPaymentProcessedKey,
			"HandleProcessed",
			logger,
		),
	)
	bus.Register(
		events.EventTypePaymentCompleted,
		handlercommon.WithIdempotency(
			payment.HandleCompleted(
				bus,
				uow,
				logger,
			),
			completedTracker,
			payment.ExtractPaymentCompletedKey,
			"HandleCompleted",
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
	bus.Register(
		events.EventTypeDepositValidated,
		deposit.HandleValidated(
			bus,
			uow,
			a.Deps.PaymentProvider,
			a.Deps.Logger,
		),
	)
}

func (a *App) setupConversionHandlers(
	bus eventbus.Bus,
	uow repository.UnitOfWork,
	exchangeRateProvider exchange.Exchange,
	logger *slog.Logger,
) {
	// 1️⃣ GENERIC CONVERSION HANDLER
	// This handler processes all conversion requests and delegates to the appropriate flow
	conversionFactories := map[string]conversion.EventFactory{
		"deposit":  &conversion.DepositEventFactory{},
		"withdraw": &conversion.WithdrawEventFactory{},
		"transfer": &conversion.TransferEventFactory{},
	}

	bus.Register(
		events.EventTypeCurrencyConversionRequested,
		conversion.HandleRequested(
			bus,
			a.Deps.ExchangeRateRegistry, // Use the exchange rate registry provider
			exchangeRateProvider,
			logger,
			conversionFactories,
		),
	)
}
