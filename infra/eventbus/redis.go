package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/eventbus"

	"github.com/redis/go-redis/v9"
)

type envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// RedisEventBus implements a production-ready event bus using Redis Streams.
type RedisEventBus struct {
	client        *redis.Client
	stream        string // Stream name per event type
	group         string // Consumer group name
	typeFactories map[string]func() common.Event
	logger        *slog.Logger
}

// NewWithRedis creates a new Redis-backed event bus.
// url: Redis connection URL (e.g., "redis://localhost:6379")
// stream: Name of the Redis stream to use
// group: Consumer group name for event processing
func NewWithRedis(url, stream, group string, types map[string]func() common.Event, logger *slog.Logger) (*RedisEventBus, error) {
	if url == "" {
		return nil, fmt.Errorf("redis event bus: url, stream, and group are required")
	}

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis event bus: invalid URL: %w", err)
	}

	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis event bus: connection failed: %w", err)
	}

	bus := &RedisEventBus{
		client:        client,
		stream:        stream,
		group:         group,
		typeFactories: types,
		logger:        logger.With("component", "redis-event-bus"),
	}

	// Initialize stream and consumer group
	ctx := context.Background()
	_ = client.XGroupCreateMkStream(ctx, stream, group, "0")
	return bus, nil
}

// Emit publishes an event to the Redis stream.
func (b *RedisEventBus) Emit(ctx context.Context, event common.Event) error {
	if b.client == nil {
		return fmt.Errorf("redis event bus: client not initialized")
	}

	b.logger.Debug("emitting event", "type", event.Type(), "event", event)

	data, err := json.Marshal(event)
	if err != nil {
		b.logger.Error("failed to marshal event", "error", err, "type", event.Type())
		return fmt.Errorf("redis event bus: marshal failed: %w", err)
	}

	env := envelope{Type: event.Type(), Payload: data}
	envBytes, err := json.Marshal(env)
	if err != nil {
		b.logger.Error("failed to marshal envelope", "error", err, "type", event.Type())
		return fmt.Errorf("redis event bus: envelope marshal failed: %w", err)
	}

	_, err = b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.stream,
		Values: map[string]any{"event": string(envBytes)},
	}).Result()

	if err != nil {
		b.logger.Error("failed to emit event", "error", err, "type", event.Type())
		return fmt.Errorf("redis event bus: emit failed: %w", err)
	}

	b.logger.Debug("event emitted successfully", "type", event.Type())
	return nil
}

// Register starts a consumer for the stream and group, calling handler for each event.
func (b *RedisEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	ctx := context.Background()
	consumer := fmt.Sprintf("consumer-%s-%d", eventType, time.Now().UnixNano())
	b.logger.Info("registering handler", "event_type", eventType, "consumer", consumer)

	go func() {
		for {
			res, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    b.group,
				Consumer: consumer,
				Streams:  []string{b.stream, ">"},
				Count:    1,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if !errors.Is(err, redis.Nil) {
					b.logger.Error("error reading from stream", "error", err, "consumer", consumer)
				}
				time.Sleep(time.Second)
				continue
			}

			for _, stream := range res {
				for _, msg := range stream.Messages {
					raw, ok := msg.Values["event"].(string)
					if !ok {
						continue
					}

					var env envelope
					if err := json.Unmarshal([]byte(raw), &env); err != nil {
						b.logger.Error("failed to unmarshal envelope", "error", err)
						continue
					}

					if env.Type != eventType {
						b.logger.Debug("Checking event type match", "got", env.Type, "expected", eventType)
						continue
					}

					constructor, ok := b.typeFactories[env.Type]
					if !ok {
						b.logger.Error("unknown event type", "event_type", env.Type)
						b.pushToDLQ(ctx, msg.Values)
						continue
					}

					evt := constructor()
					if err := json.Unmarshal(env.Payload, evt); err != nil {
						b.logger.Error("failed to unmarshal payload", "error", err, "event_type", env.Type)
						continue
					}

					func() {
						defer func() {
							if r := recover(); r != nil {
								b.logger.Error("handler panic recovered", "panic", r, "event_type", env.Type)
								b.pushToDLQ(ctx, msg.Values)
							}
						}()
						if err := handler(ctx, evt); err != nil {
							b.logger.Error("handler error", "error", err, "event_type", env.Type)
							b.pushToDLQ(ctx, msg.Values)
						}
					}()

					if err := b.client.XAck(ctx, b.stream, b.group, msg.ID).Err(); err != nil {
						b.logger.Error("failed to acknowledge message", "error", err, "msg_id", msg.ID)
					}
				}
			}
		}
	}()
	b.logger.Info("handler registered successfully", "event_type", eventType, "consumer", consumer)
}

// pushToDLQ pushes the raw event (msg.Values) to a DLQ Redis stream for inspection or reprocessing.
func (b *RedisEventBus) pushToDLQ(ctx context.Context, values map[string]any) {
	dlqStream := b.stream + "-DLQ"
	if _, err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: dlqStream,
		Values: values,
	}).Result(); err != nil {
		b.logger.Error("failed to push to DLQ", "error", err, "stream", dlqStream)
	} else {
		b.logger.Warn("event pushed to DLQ", "stream", dlqStream, "values", values)
	}
}
