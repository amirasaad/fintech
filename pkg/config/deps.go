package config

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/amirasaad/fintech/pkg/provider/payment"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/repository"
)

// Deps holds all infrastructure dependencies for building the app and services.
type Deps struct {
	Uow                          repository.UnitOfWork
	ExchangeRateProvider         exchange.Exchange
	ExchangeRateRegistryProvider registry.Provider
	CurrencyRegistry             registry.Provider
	PaymentProvider              payment.Payment
	EventBus                     eventbus.Bus
	Logger                       *slog.Logger
	Config                       *App
}
