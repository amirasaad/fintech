package initializer

import (
	"io"
	"log/slog"
	"testing"

	infra_eventbus "github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestInitEventBus_DefaultsToMemoryAsyncWhenNoExplicitDriver(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.App{
		Redis:    &config.Redis{URL: "redis://localhost:6379/0"},
		EventBus: &config.EventBus{Driver: ""},
	}

	bus, err := initEventBus(cfg, logger)
	require.NoError(t, err)
	require.IsType(t, &infra_eventbus.MemoryAsyncEventBus{}, bus)
}

func TestInitEventBus_ExplicitRedisRequiresURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.App{
		Redis:    &config.Redis{URL: ""},
		EventBus: &config.EventBus{Driver: "redis", RedisURL: ""},
	}

	_, err := initEventBus(cfg, logger)
	require.Error(t, err)
}

func TestInitEventBus_RedisConnectionErrorFallsBackToMemoryAsync(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.App{
		EventBus: &config.EventBus{Driver: "redis", RedisURL: "redis://127.0.0.1:1"},
	}

	bus, err := initEventBus(cfg, logger)
	require.NoError(t, err)
	require.IsType(t, &infra_eventbus.MemoryAsyncEventBus{}, bus)
}

func TestInitEventBus_ExplicitKafkaRequiresBrokers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.App{
		EventBus: &config.EventBus{Driver: "kafka", KafkaBrokers: ""},
	}

	_, err := initEventBus(cfg, logger)
	require.Error(t, err)
}

func TestInitEventBus_KafkaConnectionErrorFallsBackToMemoryAsync(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.App{
		EventBus: &config.EventBus{Driver: "kafka", KafkaBrokers: "127.0.0.1:1"},
	}

	bus, err := initEventBus(cfg, logger)
	require.NoError(t, err)
	require.IsType(t, &infra_eventbus.MemoryAsyncEventBus{}, bus)
}

func TestInitEventBus_UnsupportedDriverErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.App{
		EventBus: &config.EventBus{Driver: "nope"},
	}

	_, err := initEventBus(cfg, logger)
	require.Error(t, err)
}
