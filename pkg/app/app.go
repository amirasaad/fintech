package app

import (
	"log/slog"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	currencyScv "github.com/amirasaad/fintech/pkg/service/currency"
	"github.com/amirasaad/fintech/pkg/service/user"
)

// Deps contains all the dependencies needed by the SetupBus function
type Deps struct {
	Uow               repository.UnitOfWork
	CurrencyConverter money.CurrencyConverter
	CurrencyRegistry  *currency.Registry
	PaymentProvider   provider.PaymentProvider
	EventBus          eventbus.Bus
	Logger            *slog.Logger
}

type App struct {
	Deps            *Deps
	Config          *config.App
	AuthService     *auth.Service
	UserService     *user.Service
	AccountService  *account.Service
	CurrencyService *currencyScv.Service
}

func New(deps *Deps, cfg *config.App) *App {
	app := &App{
		Deps:   deps,
		Config: cfg,
	}
	app.setupEventBus()

	authMap := map[string]func() *auth.Service{
		"jwt": func() *auth.Service {
			return auth.NewWithJWT(deps.Uow, cfg.Auth.Jwt, deps.Logger)
		},
	}
	if authFactory, ok := authMap[cfg.Auth.Strategy]; ok {
		app.AuthService = authFactory()
	} else {
		app.AuthService = auth.NewWithBasic(deps.Uow, deps.Logger)
	}
	app.UserService = user.New(deps.Uow, deps.Logger)
	app.AccountService = account.New(deps.EventBus, deps.Uow, deps.Logger)
	app.CurrencyService = currencyScv.New(deps.CurrencyRegistry, deps.Logger)
	return app
}
