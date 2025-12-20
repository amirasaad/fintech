//go:build !redis
// +build !redis

package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

type RedisEventBusConfig struct {
	DLQRetryInterval  time.Duration
	DLQBatchSize      int64
	DLQMaxRetries     int
	DLQInitialBackoff time.Duration
	DLQMaxBackoff     time.Duration
}

func DefaultRedisEventBusConfig() *RedisEventBusConfig {
	return &RedisEventBusConfig{
		DLQRetryInterval: 5 * time.Minute,
		DLQBatchSize:     10,
	}
}

type RedisEventBus struct{}

func NewWithRedis(
	url string,
	logger *slog.Logger,
	config *RedisEventBusConfig,
) (*RedisEventBus, error) {
	return nil, fmt.Errorf("redis event bus: build with -tags redis to enable")
}

func (b *RedisEventBus) Register(eventType events.EventType, handler eventbus.HandlerFunc) {
}

func (b *RedisEventBus) Emit(ctx context.Context, event events.Event) error {
	return fmt.Errorf("redis event bus: build with -tags redis to enable")
}

var _ eventbus.Bus = (*RedisEventBus)(nil)
