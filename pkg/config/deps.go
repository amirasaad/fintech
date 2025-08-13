package config

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
)

// Deps holds all infrastructure dependencies for building the app and services.
type Deps struct {
	Uow               repository.UnitOfWork
	CurrencyConverter currency.Converter
	CurrencyRegistry  *currency.Registry
	PaymentProvider   provider.Payment
	EventBus          eventbus.Bus
	Logger            *slog.Logger
	Config            *App
}
