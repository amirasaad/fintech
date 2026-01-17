package initializer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/caching"
	infra_eventbus "github.com/amirasaad/fintech/infra/eventbus"
	exchangerateapi "github.com/amirasaad/fintech/infra/provider/exchangerateapi"
	stripepayment "github.com/amirasaad/fintech/infra/provider/stripepayment"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	currencyfixtures "github.com/amirasaad/fintech/internal/fixtures/currency"
	"github.com/amirasaad/fintech/pkg/app"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider/exchange"

	"github.com/amirasaad/fintech/pkg/registry"
)

// loadCurrencyFixtures loads currency metadata into the registry.
func loadCurrencyFixtures(
	ctx context.Context,
	registryProvider registry.Provider,
	logger *slog.Logger,
) {
	logger.Info("Loading currency metadata")
	_, filename, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(
		filepath.Dir(filename),
		"../../internal/fixtures/currency/meta.csv",
	)

	source := "embedded"
	var entities []registry.Entity
	var err error

	if _, statErr := os.Stat(fixturePath); statErr == nil {
		entities, err = currencyfixtures.LoadCurrencyMetaCSV(fixturePath)
		source = fixturePath
	} else if !os.IsNotExist(statErr) {
		logger.Warn("Failed to stat currency meta CSV", "path", fixturePath, "error", statErr)
	}

	if source == "embedded" {
		entities, err = currencyfixtures.LoadCurrencyMetaCSV("")
	}
	if err != nil {
		logger.Warn("Failed to load currency meta from fixture", "source", source, "error", err)
		return
	}

	logger.Info("Loading currency meta from fixture",
		"source", source,
		"to_register", len(entities))

	var registeredCount int
	for _, entity := range entities {
		if err := registryProvider.Register(ctx, entity); err != nil {
			logger.Error("Failed to register currency", "code", entity.ID(), "error", err)
			// Continue with other currencies even if one fails
		} else {
			registeredCount++
		}
	}

	logger.Info("Successfully loaded currency fixtures", "registered_count", registeredCount)
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

	// Initialize registry providers for each service
	deps.RegistryProvider, err = GetDefaultRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize main registry provider: %w", err)
	}

	// Initialize currency registry with dedicated provider
	deps.CurrencyRegistry, err = GetCurrencyRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize currency registry provider: %w", err)
	}

	ctx := context.Background()
	// Only load currency fixtures if the registry is empty
	count, err := deps.CurrencyRegistry.Count(ctx)
	if err != nil {
		logger.Warn("Failed to check currency registry count", "error", err)
	}

	if count == 0 {
		loadCurrencyFixtures(ctx, deps.CurrencyRegistry, logger)
	} else {
		logger.Info("Skipping currency fixtures load; registry not empty", "existing_count", count)
	}

	// Initialize checkout registry
	deps.CheckoutRegistry, err = GetCheckoutRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize checkout registry provider: %w", err)
	}

	// Initialize exchange rate registry
	deps.ExchangeRateRegistry, err = GetExchangeRateRegistry(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exchange rate registry provider: %w", err)
	}

	// Create the exchange rate provider
	exchangeProvider := exchangerateapi.NewExchangeRateAPIProvider(
		cfg.ExchangeRateAPIProviders.ExchangeRateApi,
		logger,
	)
	deps.ExchangeRateProvider = exchangeProvider

	// Initialize exchange rates
	if eerr := initializeExchangeRates(
		ctx,
		exchangeProvider,
		deps.ExchangeRateRegistry,
		cfg.ExchangeRateCache,
		logger,
	); eerr != nil {
		logger.Error("Failed to initialize exchange rates", "error", eerr)
		// Don't fail the entire startup for exchange rate initialization
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
	bus, err := initEventBus(cfg, logger)
	if err != nil {
		return nil, err
	}
	deps.EventBus = bus

	// Initialize payment provider with the checkout registry and unit of work
	deps.PaymentProvider = stripepayment.New(
		bus,
		deps.CheckoutRegistry, // Use the checkout-specific registry
		cfg.PaymentProviders.Stripe,
		logger,
		deps.Uow, // Pass the repository's UnitOfWork
	)

	return
}

func initEventBus(cfg *config.App, logger *slog.Logger) (eventbus.Bus, error) {
	explicitDriver := ""
	if cfg.EventBus != nil {
		explicitDriver = strings.TrimSpace(cfg.EventBus.Driver)
	}

	if explicitDriver == "" {
		return infra_eventbus.NewWithMemoryAsync(logger), nil
	}

	driver := strings.TrimSpace(strings.ToLower(explicitDriver))
	switch driver {
	case "memory":
		return infra_eventbus.NewWithMemoryAsync(logger), nil
	case "redis":
		redisURL := ""
		if cfg.EventBus != nil {
			redisURL = strings.TrimSpace(cfg.EventBus.RedisURL)
		}
		if redisURL == "" && cfg.Redis != nil {
			redisURL = strings.TrimSpace(cfg.Redis.URL)
		}
		if redisURL == "" {
			return nil, fmt.Errorf("event bus redis: redis url is required")
		}
		busConfig := &infra_eventbus.RedisEventBusConfig{
			DLQRetryInterval: 5 * time.Minute,
			DLQBatchSize:     10,
		}
		bus, err := infra_eventbus.NewWithRedis(redisURL, logger, busConfig)
		if err != nil {
			logger.Warn("Redis event bus init failed, falling back to memory async", "error", err)
			return infra_eventbus.NewWithMemoryAsync(logger), nil
		}
		return bus, nil
	case "kafka":
		if cfg.EventBus == nil {
			return nil, fmt.Errorf("event bus kafka: configuration is required")
		}
		brokers := strings.TrimSpace(cfg.EventBus.KafkaBrokers)
		if brokers == "" {
			return nil, fmt.Errorf("event bus kafka: brokers are required")
		}
		caFilePath, err := ensureKafkaCAFile(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("event bus kafka: prepare tls ca file: %w", err)
		}
		certFilePath, err := ensureKafkaTLSCertFile(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("event bus kafka: prepare tls cert file: %w", err)
		}
		keyFilePath, err := ensureKafkaTLSKeyFile(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("event bus kafka: prepare tls key file: %w", err)
		}
		tlsCertSet := strings.TrimSpace(certFilePath) != ""
		tlsKeySet := strings.TrimSpace(keyFilePath) != ""
		if cfg.EventBus.KafkaTLSEnabled && tlsCertSet != tlsKeySet {
			return nil, fmt.Errorf("event bus kafka: tls cert and key must be provided together")
		}
		saslUsernameSet := strings.TrimSpace(cfg.EventBus.KafkaSASLUsername) != ""
		saslPasswordSet := strings.TrimSpace(cfg.EventBus.KafkaSASLPassword) != ""
		tlsCaProvided := strings.TrimSpace(caFilePath) != ""
		tlsInputsProvided := tlsCaProvided ||
			tlsCertSet ||
			tlsKeySet ||
			cfg.EventBus.KafkaTLSSkipVerify

		brokerCount := 0
		for _, broker := range strings.Split(brokers, ",") {
			if strings.TrimSpace(broker) != "" {
				brokerCount++
			}
		}

		if !cfg.EventBus.KafkaTLSEnabled && tlsInputsProvided {
			logger.Warn("Kafka TLS settings provided but TLS disabled",
				"tls_enabled", cfg.EventBus.KafkaTLSEnabled,
				"tls_ca_file", strings.TrimSpace(caFilePath),
				"tls_cert_file_set", tlsCertSet,
				"tls_key_file_set", tlsKeySet,
				"tls_skip_verify", cfg.EventBus.KafkaTLSSkipVerify,
			)
		}
		logger.Info("Initializing Kafka event bus",
			"brokers", brokers,
			"brokers_count", brokerCount,
			"group_id", strings.TrimSpace(cfg.EventBus.KafkaGroupID),
			"topic_prefix", strings.TrimSpace(cfg.EventBus.KafkaTopic),
			"tls_enabled", cfg.EventBus.KafkaTLSEnabled,
			"tls_ca_file", strings.TrimSpace(caFilePath),
			"tls_cert_file_set", tlsCertSet,
			"tls_key_file_set", tlsKeySet,
			"tls_skip_verify", cfg.EventBus.KafkaTLSSkipVerify,
			"sasl_username_set", saslUsernameSet,
			"sasl_password_set", saslPasswordSet,
		)
		kafkaConfig := &infra_eventbus.KafkaEventBusConfig{
			GroupID:          strings.TrimSpace(cfg.EventBus.KafkaGroupID),
			TopicPrefix:      strings.TrimSpace(cfg.EventBus.KafkaTopic),
			DLQRetryInterval: 5 * time.Minute,
			DLQBatchSize:     10,
			SASLUsername:     strings.TrimSpace(cfg.EventBus.KafkaSASLUsername),
			SASLPassword:     strings.TrimSpace(cfg.EventBus.KafkaSASLPassword),
			TLSEnabled:       cfg.EventBus.KafkaTLSEnabled,
			TLSCAFile:        strings.TrimSpace(caFilePath),
			TLSCertFile:      strings.TrimSpace(certFilePath),
			TLSKeyFile:       strings.TrimSpace(keyFilePath),
			TLSSkipVerify:    cfg.EventBus.KafkaTLSSkipVerify,
		}
		bus, err := infra_eventbus.NewWithKafka(brokers, logger, kafkaConfig)
		if err != nil {
			logger.Warn("Kafka event bus init failed, falling back to memory async", "error", err)
			return infra_eventbus.NewWithMemoryAsync(logger), nil
		}
		return bus, nil
	default:
		return nil, fmt.Errorf("unsupported event bus driver: %s", driver)
	}
}

func ensureKafkaTLSFile(
	pem string,
	tempPattern string,
	pemLabel string,
	fileLabel string,
	logMessage string,
	logger *slog.Logger,
) (string, error) {
	pem = strings.TrimSpace(pem)
	if pem == "" {
		return "", nil
	}

	pem = strings.ReplaceAll(pem, "\\n", "\n")
	if strings.TrimSpace(pem) == "" {
		return "", fmt.Errorf("%s is empty", pemLabel)
	}

	tmpfile, err := os.CreateTemp("", tempPattern)
	if err != nil {
		return "", fmt.Errorf("create temp %s file: %w", fileLabel, err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("close temp %s file: %w", fileLabel, err)
	}
	path := tmpfile.Name()

	if err := os.WriteFile(path, []byte(pem), 0600); err != nil {
		return "", fmt.Errorf("write %s file: %w", fileLabel, err)
	}
	if logger != nil && logMessage != "" {
		logger.Info(logMessage, "path", path)
	}

	return path, nil
}

func ensureKafkaCAFile(cfg *config.App, logger *slog.Logger) (string, error) {
	if cfg == nil || cfg.EventBus == nil {
		return "", nil
	}

	return ensureKafkaTLSFile(
		cfg.EventBus.KafkaTLSCAPem,
		"fintech-kafka-ca-*.pem",
		"kafka ca pem",
		"kafka ca",
		"Kafka CA file written",
		logger,
	)
}

func ensureKafkaTLSCertFile(cfg *config.App, logger *slog.Logger) (string, error) {
	if cfg == nil || cfg.EventBus == nil {
		return "", nil
	}

	return ensureKafkaTLSFile(
		cfg.EventBus.KafkaTLSCertPem,
		"fintech-kafka-cert-*.pem",
		"kafka tls cert pem",
		"kafka tls cert",
		"Kafka TLS cert file written",
		logger,
	)
}

func ensureKafkaTLSKeyFile(cfg *config.App, logger *slog.Logger) (string, error) {
	if cfg == nil || cfg.EventBus == nil {
		return "", nil
	}

	return ensureKafkaTLSFile(
		cfg.EventBus.KafkaTLSKeyPem,
		"fintech-kafka-key-*.pem",
		"kafka tls key pem",
		"kafka tls key",
		"Kafka TLS key file written",
		logger,
	)
}

// initializeExchangeRates fetches and caches exchange rates during application startup
// and sets up a background refresh mechanism
func initializeExchangeRates(
	ctx context.Context,
	exchangeRateProvider exchange.Exchange,
	registryProvider registry.Provider,
	cfg *config.ExchangeRateCache,
	logger *slog.Logger,
) error {
	// Start the background refresh goroutine
	go func(ctx context.Context, cacheLogger *slog.Logger) {
		// Initialize the exchange cache with the provided registry provider and config
		exchangeCache := caching.NewExchangeCache(
			registryProvider,
			cacheLogger,
			cfg,
		)

		// Set up periodic refresh
		ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Check if cache is stale before refreshing
				isStale, timeUntilRefresh, err := exchangeCache.IsCacheStale(ctx)
				if err != nil {
					logger.Warn("Failed to check cache staleness", "error", err)
					continue
				}
				logger.Debug(
					"Cache staleness check",
					"is_stale", isStale,
					"time_until_refresh", timeUntilRefresh,
				)

				if isStale {
					logger.Info("Cache is stale, refreshing exchange rates")
					if err := refreshExchangeRates(
						ctx, exchangeRateProvider, exchangeCache, logger); err != nil {
						logger.Error("Failed to refresh exchange rates", "error", err)
					} else {
						logger.Info("Successfully refreshed exchange rates")
					}
				} else {
					logger.Debug(
						"Cache is still fresh, next refresh in",
						"duration", timeUntilRefresh,
					)
				}

			case <-ctx.Done():
				return
			}
		}
	}(ctx, logger)

	return nil
}

// refreshExchangeRates handles the actual refreshing of exchange rates
func refreshExchangeRates(
	ctx context.Context,
	exchangeRateProvider exchange.Exchange,
	exchangeCache *caching.ExchangeCache,
	logger *slog.Logger,
) error {
	// Create a timeout context for the refresh operation
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch rates from the provider
	rates, err := exchangeRateProvider.FetchRates(
		ctx,
		"USD", // Base currency
	)
	if err != nil {
		return fmt.Errorf("failed to fetch exchange rates: %w", err)
	}

	// Cache the rates using ExchangeCache
	if err := exchangeCache.CacheRates(
		ctx,
		rates,
		exchangeRateProvider.Metadata().Name,
	); err != nil {
		return fmt.Errorf("failed to cache exchange rates: %w", err)
	}

	logger.Info("Successfully cached exchange rates",
		"provider", exchangeRateProvider.Metadata().Name,
		"rates_count", len(rates),
	)
	return nil

}
