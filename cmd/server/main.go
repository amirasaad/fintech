package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"gorm.io/gorm"

	"github.com/amirasaad/fintech/webapi"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// @title Fintech API
// @version 1.0.0
// @description Fintech API documentation
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email fiber@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/MIT
// @host localhost:3000
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Enter your Bearer token in the format: `Bearer {token}`"
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	logger := setupLogger()

	cfg, err := loadConfig(logger)
	if err != nil {
		return err
	}

	currencyRegistry, err := initCurrencyRegistry(cfg, logger)
	if err != nil {
		return err
	}

	checkoutRegistry, err := initCheckoutRegistry(cfg, logger)
	if err != nil {
		return err
	}

	db, err := initDatabase(cfg, logger)
	if err != nil {
		return err
	}

	uow := infra_repository.NewUoW(db)

	currencyConverter, err := initExchangeSystem(cfg, logger, currencyRegistry)
	if err != nil {
		return err
	}

	bus, err := initEventBus(cfg, logger)
	if err != nil {
		return err
	}

	logger.Info(
		"Starting fintech server",
		"port", ":3000",
		"redis_configured", cfg.Redis.URL != "")

	return webapi.SetupApp(app.New(app.Deps{
		Uow:               uow,
		EventBus:          bus,
		CurrencyConverter: currencyConverter,
		CurrencyRegistry:  currencyRegistry,
		PaymentProvider: provider.NewStripePaymentProvider(
			bus,
			&cfg.PaymentProviders.Stripe,
			checkout.NewService(checkoutRegistry),
			logger,
		),
		Logger: logger,
	}, cfg)).Listen(
		fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
}

func setupLogger() *slog.Logger {
	handler := log.NewWithOptions(os.Stdout, log.Options{
		ReportTimestamp: true,
		Level:           log.DebugLevel,
		TimeFunction:    log.NowUTC,
		TimeFormat:      time.Kitchen,
		ReportCaller:    true,
	})
	styles := log.DefaultStyles()
	// Iceberg theme colors
	errorTxtColor := lipgloss.AdaptiveColor{Light: "#e78284", Dark: "#e78284"} // Red
	infoTxtColor := lipgloss.AdaptiveColor{Light: "#8caaee", Dark: "#8caaee"}  // Blue
	warnTxtColor := lipgloss.AdaptiveColor{Light: "#e5c890", Dark: "#e5c890"}  // Yellow
	debugTxtColor := lipgloss.AdaptiveColor{Light: "#ca9ee6", Dark: "#ca9ee6"} // Purple

	// Error level styling
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("❌ ERROR").
		Bold(true).
		Padding(0, 1).
		Foreground(errorTxtColor)

	// Info level styling
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("ℹ️  INFO").
		Bold(true).
		Padding(0, 1).
		Foreground(infoTxtColor)

	// Warn level styling
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("⚠️  WARN").
		Bold(true).
		Padding(0, 1).
		Foreground(warnTxtColor)

	// Debug level styling
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		SetString("🐛 DEBUG").
		Bold(true).
		Padding(0, 1).
		Foreground(debugTxtColor)

	// Error key-value styling
	styles.Keys["error"] = lipgloss.NewStyle().Foreground(errorTxtColor)
	styles.Values["error"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(errorTxtColor)
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)
	handler.SetStyles(styles)
	// Setup structured logging
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func loadConfig(logger *slog.Logger) (*config.AppConfig, error) {
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		logger.Error("Failed to load application configuration", "error", err)
		return cfg, err
	}
	logger.Info("Configuration loaded successfully",
		"database_url_configured", cfg.DB.Url != "",
		"jwt_expiry", cfg.Jwt.Expiry,
		"exchange_rate_api_configured", cfg.Exchange.ApiKey != "")
	return cfg, nil
}

func initCurrencyRegistry(
	cfg *config.AppConfig,
	logger *slog.Logger,
) (*currency.Registry, error) {
	ctx := context.Background()
	if cfg.Redis.URL != "" {
		redisURL := cfg.Redis.URL
		keyPrefix := cfg.Redis.KeyPrefix + "currency:"
		if err := currency.InitializeGlobalRegistry(ctx, redisURL, keyPrefix); err != nil {
			logger.Error("Failed to initialize global currency registry with Redis",
				"error", err,
				"redis_url", cfg.Redis.URL,
				"key_prefix", cfg.Redis.KeyPrefix)
			return nil, err
		}
	} else {
		if err := currency.InitializeGlobalRegistry(ctx); err != nil {
			logger.Error(
				"Failed to initialize global currency registry",
				"error", err,
				"redis_configured", cfg.Redis.URL != "")
			return nil, err
		}
	}
	currencyRegistry := currency.GetGlobalRegistry()
	if cfg.Redis.URL != "" {
		logger.Info(
			"Currency registry initialized with Redis cache",
			"redis_url", cfg.Redis.URL,
			"key_prefix", cfg.Redis.KeyPrefix)
	} else {
		logger.Info(
			"Currency registry initialized with in-memory cache",
			"redis_configured", cfg.Redis.URL != "")
	}
	return currencyRegistry, nil
}

func initCheckoutRegistry(
	cfg *config.AppConfig,
	logger *slog.Logger) (registry.Provider, error) {
	checkoutRegistry, err := registry.NewBuilder().
		WithName("checkout").
		WithRedis(cfg.Redis.URL).
		WithKeyPrefix(cfg.Redis.KeyPrefix+"checkout:").
		WithCache(1000, 15*time.Minute).
		BuildRegistry()
	if err != nil {
		logger.Error(
			"Failed to create checkout registry",
			"error", err,
			"redis_configured", cfg.Redis.URL != "")
		return nil, err
	}
	if cfg.Redis.URL != "" {
		logger.Info(
			"Checkout registry initialized with Redis cache",
			"redis_url", cfg.Redis.URL,
			"key_prefix", cfg.Redis.KeyPrefix)
	} else {
		logger.Info(
			"Checkout registry initialized with in-memory cache",
			"redis_configured", cfg.Redis.URL != "")
	}
	return checkoutRegistry, nil
}

func initDatabase(
	cfg *config.AppConfig,
	logger *slog.Logger) (*gorm.DB, error) {
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error(
			"Failed to initialize database",
			"error", err,
			"database_url_configured", cfg.DB.Url != "")
		return nil, err
	}
	return db, nil
}

func initExchangeSystem(
	cfg *config.AppConfig,
	logger *slog.Logger,
	currencyRegistry *currency.Registry,
) (money.CurrencyConverter, error) {
	currencyConverter, err := infra.NewExchangeRateSystem(logger, cfg.Exchange)
	if err != nil {
		logger.Error(
			"Failed to initialize exchange rate system",
			"error", err,
			"exchange_rate_api_configured", cfg.Exchange.ApiKey != "")
		return nil, err
	}
	return currencyConverter, nil
}

func initEventBus(cfg *config.AppConfig, logger *slog.Logger) (*eventbus.RedisEventBus, error) {
	bus, err := eventbus.NewWithRedis(
		cfg.Redis.URL,
		logger,
	)
	if err != nil {
		logger.Error(
			"Failed to initialize event bus",
			"error", err,
			"redis_configured", cfg.Redis.URL != "")
		return nil, err
	}
	return bus, nil
}
