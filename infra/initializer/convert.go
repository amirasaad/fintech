package initializer

import (
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/domain/money"
)

// ToAppDeps converts initializer.Deps to app.Deps
func (d *Deps) ToAppDeps(currencyConverter money.CurrencyConverter) app.Deps {
	return app.Deps{
		Uow:               d.Uow,
		EventBus:          d.EventBus,
		CurrencyRegistry:  d.CurrencyRegistry,
		PaymentProvider:   d.PaymentProvider,
		Logger:            d.Logger,
		CurrencyConverter: currencyConverter,
	}
}
