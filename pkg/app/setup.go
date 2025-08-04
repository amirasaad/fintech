// Package app provides functionality for setting up and configuring the event Bus
// with all necessary event handlers for the application.
package app

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/amirasaad/fintech/pkg/handler/conversion"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
)

// Dependencies contains all the dependencies needed by the SetupBus function
type Dependencies struct {
	Bus               eventbus.Bus
	Uow               repository.UnitOfWork
	CurrencyConverter money.CurrencyConverter
	PaymentProvider   provider.PaymentProvider
	Logger            *slog.Logger
}

// SetupBus registers all event handlers with the provided event Bus.
func SetupBus(deps Dependencies) {
	// 1️⃣ GENERIC CONVERSION HANDLER
	// Handles all ConversionRequestedEvent by delegating to a flow-specific factory.
	conversionFactories := map[string]conversion.EventFactory{
		"deposit":  &conversion.DepositEventFactory{},
		"withdraw": &conversion.WithdrawEventFactory{},
		"transfer": &conversion.TransferEventFactory{},
	}
	bus := deps.Bus
	bus.Register(
		events.EventTypeCurrencyConversionRequested,
		conversion.HandleRequested(
			bus,
			deps.CurrencyConverter,
			deps.Logger,
			conversionFactories,
		),
	)
	bus.Register(
		events.EventTypeCurrencyConverted,
		conversion.HandleCurrencyConverted(
			deps.Uow,
			deps.Logger,
		),
	)

	bus.Register(
		events.EventTypeDepositRequested,
		deposit.HandleRequested(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)
	bus.Register(
		events.EventTypeDepositCurrencyConverted,
		deposit.HandleCurrencyConverted(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)

	bus.Register(
		events.EventTypeWithdrawRequested,
		withdraw.HandleRequested(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)
	bus.Register(
		events.EventTypeWithdrawCurrencyConverted,
		withdraw.HandleCurrencyConverted(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)

	bus.Register(
		events.EventTypePaymentInitiated,
		payment.HandleInitiated(
			bus,
			deps.PaymentProvider,
			deps.Logger,
		),
	)
	bus.Register(
		events.EventTypePaymentProcessed,
		payment.HandleProcessed(
			deps.Uow,
			deps.Logger,
		),
	)
	bus.Register(
		events.EventTypePaymentCompleted,
		payment.HandleCompleted(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)

	bus.Register(
		events.EventTypeTransferRequested,
		transfer.HandleRequested(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)
	bus.Register(
		events.EventTypeTransferCurrencyConverted,
		transfer.HandleCurrencyConverted(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)
	bus.Register(
		events.EventTypeTransferCompleted,
		transfer.HandleCompleted(
			bus,
			deps.Uow,
			deps.Logger,
		),
	)
}
