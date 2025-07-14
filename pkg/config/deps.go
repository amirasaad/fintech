package config

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
)

// Deps holds all infrastructure dependencies for building the app and services.
type Deps struct {
	Uow               repository.UnitOfWork
	CurrencyConverter money.CurrencyConverter
	PaymentProvider   provider.PaymentProvider
	EventBus          eventbus.EventBus
	CurrencyRegistry  *currency.CurrencyRegistry
	Converter         money.CurrencyConverter
	Logger            *slog.Logger
	Config            *AppConfig
}
