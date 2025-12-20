//go:build !kafka
// +build !kafka

package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

type KafkaEventBusConfig struct {
	GroupID          string
	TopicPrefix      string
	DLQRetryInterval time.Duration
	DLQBatchSize     int
}

type KafkaEventBus struct{}

func NewWithKafka(
	brokers string,
	logger *slog.Logger,
	config *KafkaEventBusConfig,
) (*KafkaEventBus, error) {
	return nil, fmt.Errorf("kafka event bus: build with -tags kafka to enable")
}

func (b *KafkaEventBus) Register(eventType events.EventType, handler eventbus.HandlerFunc) {
}

func (b *KafkaEventBus) Emit(ctx context.Context, event events.Event) error {
	return fmt.Errorf("kafka event bus: build with -tags kafka to enable")
}

var _ eventbus.Bus = (*KafkaEventBus)(nil)
