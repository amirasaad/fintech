package initializer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amirasaad/fintech/infra"
	infra_eventbus "github.com/amirasaad/fintech/infra/eventbus"
	infra_provider "github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Deps contains all the dependencies needed by the application
type Deps struct {
	Uow              repository.UnitOfWork
	EventBus         eventbus.Bus
	CurrencyRegistry *currency.Registry
	PaymentProvider  provider.PaymentProvider
	Logger           *slog.Logger
}

// InitializeDependencies initializes all the application dependencies
func InitializeDependencies(cfg *config.App) (
	deps *app.Deps,
	err error,
) {
	// Load configuration
	deps = &app.Deps{}
	logger := setupLogger(cfg.Log)
	deps.Logger = logger
	// Initialize currency registry
	deps.CurrencyRegistry, err = initCurrencyRegistry(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Initialize checkout registry
	checkoutRegistry, err := initCheckoutRegistry(cfg, logger)
	if err != nil {
		return nil, err
	}

	// Initialize database
	db, err := infra.NewDBConnection(cfg.DB, cfg.Env)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		return nil, err
	}

	// Initialize unit of work
	deps.Uow = infra_repository.NewUoW(db)

	// Initialize event bus
	var bus eventbus.Bus
	if cfg.Redis.URL != "" {
		redisBus, err := infra_eventbus.NewWithRedis(cfg.Redis.URL, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Redis event bus: %w", err)
		}
		bus = redisBus
	} else {
		// Fall back to in-memory event bus if Redis is not configured
		bus = infra_eventbus.NewWithMemory(logger)
	}

	deps.EventBus = bus

	// Initialize payment provider
	deps.PaymentProvider = infra_provider.NewStripePaymentProvider(
		bus,
		cfg.PaymentProviders.Stripe,
		checkout.NewService(checkoutRegistry),
		logger,
	)

	return
}

func initCurrencyRegistry(cfg *config.App, logger *slog.Logger) (*currency.Registry, error) {
	ctx := context.Background()
	keyPrefix := cfg.Redis.KeyPrefix + "currency:"
	if err := currency.InitializeGlobalRegistry(ctx, cfg.Redis.URL, keyPrefix); err != nil {
		logger.Error("Failed to initialize global currency registry with Redis",
			"error", err,
			"redis_url", cfg.Redis.URL,
			"key_prefix", keyPrefix)
		return nil, err
	}
	logger.Info("Currency registry initialized with Redis cache",
		"redis_url", cfg.Redis.URL,
		"key_prefix", keyPrefix)

	return currency.GetGlobalRegistry(), nil
}

func initCheckoutRegistry(cfg *config.App, logger *slog.Logger) (registry.Provider, error) {
	checkoutRegistry, err := registry.NewBuilder().
		WithName("checkout").
		WithRedis(cfg.Redis.URL).
		WithKeyPrefix(cfg.Redis.KeyPrefix+"checkout:").
		WithCache(1000, 15*time.Minute).
		BuildRegistry()
	if err != nil {
		logger.Error("Failed to create checkout registry",
			"error", err,
			"redis_configured", cfg.Redis.URL != "")
		return nil, err
	}
	return checkoutRegistry, nil
}

func setupLogger(cfg *config.Log) *slog.Logger {
	// Create a new logger with a custom style
	// Define color styles for different log levels
	styles := log.DefaultStyles()
	infoTxtColor := lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	warnTxtColor := lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	errorTxtColor := lipgloss.AdaptiveColor{Light: "#FF6B6B", Dark: "#FF6B6B"}
	debugTxtColor := lipgloss.AdaptiveColor{Light: "#7E57C2", Dark: "#7E57C2"}

	// Customize the style for each log level
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

	// Create a new logger with the custom styles
	logger := log.NewWithOptions(os.Stdout, log.Options{
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      cfg.TimeFormat,
		Level:           log.DebugLevel,
		Prefix:          cfg.Prefix,
	})

	logger.SetStyles(styles) // Convert to slog.Logger
	slogger := slog.New(logger)
	slog.SetDefault(slogger)

	return slogger
}
