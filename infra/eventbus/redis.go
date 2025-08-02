package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/redis/go-redis/v9"
)

type envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// RedisEventBus implements a production-ready event bus using Redis Streams.
type RedisEventBus struct {
	client      *redis.Client
	handlers    map[string][]eventbus.HandlerFunc // Map of event types to their handlers
	handlersMtx sync.RWMutex
	logger      *slog.Logger
}

const streamPrefix = "events"

// NewWithRedis creates a new Redis-backed event bus.
// url: Redis connection URL (e.g., "redis://localhost:6379")
func NewWithRedis(url string, logger *slog.Logger) (*RedisEventBus, error) {
	if url == "" {
		return nil, fmt.Errorf("redis event bus: url is required")
	}
	client, err := setupRedisClient(url)
	if err != nil {
		return nil, err
	}
	bus := createRedisEventBus(client, logger)
	return bus, nil
}

// Emit publishes an event to the Redis stream.
func (b *RedisEventBus) Emit(ctx context.Context, event events.Event) error {
	if err := b.validateClient(); err != nil {
		return err
	}
	envBytes, err := b.buildEnvelope(event)
	if err != nil {
		return err
	}
	if err := b.publishEnvelope(ctx, event.Type(), envBytes); err != nil {
		return err
	}
	b.logger.Debug("event emitted successfully", "type", event.Type())
	return nil
}

// Register registers an event handler for a specific event type.
func (b *RedisEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	b.registerHandler(eventType, handler)
	if err := b.startConsumerForEvent(eventType); err != nil {
		b.logger.Error("failed to register event handler", "type", eventType, "err", err)
		return
	}
}

// createRedisEventBus initializes the RedisEventBus struct.
func createRedisEventBus(client *redis.Client, logger *slog.Logger) *RedisEventBus {
	return &RedisEventBus{
		client:   client,
		handlers: make(map[string][]eventbus.HandlerFunc),
		logger:   logger.With("bus", "redis"),
	}
}

// streamFor returns the stream name for a given event type.
func (b *RedisEventBus) streamFor(eventType string) string {
	return fmt.Sprintf("%s:%s", streamPrefix, eventType)
}

// initializeConsumerGroup ensures group exists and cleans up idle consumers.
func (b *RedisEventBus) initializeConsumerGroup(eventType string) error {
	ctx := context.Background()
	group := b.groupNameFor(eventType)
	stream := b.streamFor(eventType)
	if err := b.ensureConsumerGroup(ctx, stream, group); err != nil {
		return err
	}
	b.cleanupIdleConsumers(ctx, stream, group)
	return nil
}

// validateClient ensures the Redis client is initialized.
func (b *RedisEventBus) validateClient() error {
	if b.client == nil {
		return fmt.Errorf("redis event bus: client not initialized")
	}
	return nil
}

// startConsumerForEvent derives group/consumer names and starts consuming for eventType.
func (b *RedisEventBus) startConsumerForEvent(eventType string) error {
	group := b.groupNameFor(eventType)
	consumer := b.consumerNameFor(eventType)
	if err := b.initializeConsumerGroup(eventType); err != nil {
		return err
	}
	stream := b.streamFor(eventType)
	b.startConsuming(eventType, stream, group, consumer)
	return nil
}

// groupNameFor returns the consumer group name for a given event type.
func (b *RedisEventBus) groupNameFor(eventType string) string {
	return fmt.Sprintf("group:%s", eventType)
}

// consumerNameFor returns the consumer name for a given event type.
func (b *RedisEventBus) consumerNameFor(eventType string) string {
	return fmt.Sprintf("consumer:%s", eventType)
}

// setupRedisClient parses URL, creates client, and pings redis.
func setupRedisClient(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis event bus: invalid URL: %w", err)
	}

	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis event bus: connection failed: %w", err)
	}
	return client, nil
}

// ensureConsumerGroup creates the group if not exists.
func (b *RedisEventBus) ensureConsumerGroup(ctx context.Context, stream, group string) error {
	err := b.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// cleanupIdleConsumers removes idle consumers with pending messages.
func (b *RedisEventBus) cleanupIdleConsumers(ctx context.Context, stream, group string) {
	if consumers, err := b.client.XInfoConsumers(ctx, stream, group).Result(); err == nil {
		for _, consumer := range consumers {
			idleDuration := time.Duration(consumer.Idle) * time.Millisecond
			if consumer.Pending > 0 && idleDuration > 5*time.Minute {
				b.client.XGroupDelConsumer(ctx, stream, group, consumer.Name)
			}
		}
	}
}

// buildEnvelope marshals event and wraps in envelope.
func (b *RedisEventBus) buildEnvelope(event events.Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		b.logger.Error("failed to marshal event", "error", err, "type", event.Type())
		return nil, fmt.Errorf("redis event bus: marshal failed: %w", err)
	}

	env := envelope{Type: event.Type(), Payload: data}
	envBytes, err := json.Marshal(env)
	if err != nil {
		b.logger.Error("failed to marshal envelope", "error", err, "type", event.Type())
		return nil, fmt.Errorf("redis event bus: envelope marshal failed: %w", err)
	}

	return envBytes, nil
}

// publishEnvelope adds the envelope to the redis stream.
func (b *RedisEventBus) publishEnvelope(ctx context.Context, eventType string, envelopeData []byte) error {
	args := b.prepareAddArgs(eventType, envelopeData)
	_, err := b.client.XAdd(ctx, args).Result()
	if err != nil {
		b.logger.Error("failed to emit event", "error", err)
		return fmt.Errorf("redis event bus: emit failed: %w", err)
	}
	return nil
}

// prepareAddArgs prepares the XAddArgs for publishing an envelope.
func (b *RedisEventBus) prepareAddArgs(eventType string, data []byte) *redis.XAddArgs {
	return &redis.XAddArgs{
		Stream: b.streamFor(eventType),
		Values: map[string]any{"event": string(data)},
	}
}

// registerHandler safely registers a handler for the given event type.
func (b *RedisEventBus) registerHandler(eventType string, handler eventbus.HandlerFunc) {
	b.handlersMtx.Lock()
	defer b.handlersMtx.Unlock()

	b.ensureHandlersMap()
	b.handlers[eventType] = append(b.handlers[eventType], handler)

	b.logger.Info("registered handler", "event_type", eventType)
}

// ensureHandlersMap initializes the handlers map if it is nil.
func (b *RedisEventBus) ensureHandlersMap() {
	if b.handlers == nil {
		b.handlers = make(map[string][]eventbus.HandlerFunc)
	}
}

// startConsuming starts a goroutine to consume events for the given eventType.
func (b *RedisEventBus) startConsuming(eventType, stream, group, consumer string) {
	go b.consume(eventType, stream, group, consumer)
}

// consume starts consuming messages from the Redis stream and routes them to the appropriate handlers.
func (b *RedisEventBus) consume(eventType, stream, group, consumer string) {
	ctx := context.Background()
	b.logger.Info("starting consumer", "group", group, "stream", stream, "event_type", eventType, "consumer", consumer)

	for {
		messages, err := b.readStream(ctx, stream, group, consumer)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		for _, msg := range messages {
			b.processMessage(ctx, eventType, group, msg)
		}
	}
}

// readStream reads messages from a redis stream group.
func (b *RedisEventBus) readStream(ctx context.Context, stream, group, consumer string) ([]redis.XMessage, error) {
	res, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    10,
		Block:    5 * time.Second,
		NoAck:    false,
	}).Result()

	if err != nil {
		if !errors.Is(err, redis.Nil) {
			b.logger.Error("error reading from stream", "error", err, "group", group)
		}
		return nil, err
	}

	var messages []redis.XMessage
	for _, stream := range res {
		messages = append(messages, stream.Messages...)
	}
	return messages, nil
}

// processMessage processes a single message from the Redis stream.
func (b *RedisEventBus) processMessage(ctx context.Context, eventType, group string, msg redis.XMessage) {
	raw, ok := msg.Values["event"].(string)
	if !ok {
		b.logger.Error("invalid message format, missing 'event' field", "msg_id", msg.ID)
		return
	}

	var env envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		b.logger.Error("failed to unmarshal envelope", "error", err, "msg_id", msg.ID)
		return
	}

	constructor, ok := events.EventTypes[env.Type]
	if !ok {
		b.logger.Error("unknown event type", "event_type", env.Type, "msg_id", msg.ID)
		b.pushToDLQ(ctx, env.Type, msg.Values)
		return
	}

	evt := constructor()
	if err := json.Unmarshal(env.Payload, evt); err != nil {
		b.logger.Error("failed to unmarshal payload", "error", err, "event_type", env.Type, "msg_id", msg.ID)
		return
	}

	handlers := b.getHandlers(env.Type)
	if len(handlers) == 0 {
		b.logger.Warn("no handlers registered for event type", "event_type", env.Type, "msg_id", msg.ID)
		_ = b.ackMessage(ctx, env.Type, group, msg.ID)
		return
	}

	success := b.executeHandlers(ctx, env.Type, evt, msg.ID, handlers)

	if success {
		if err := b.ackMessage(ctx, env.Type, group, msg.ID); err != nil {
			b.logger.Error("failed to ack message", "error", err, "msg_id", msg.ID)
		}
	} else {
		b.logger.Warn("sending message to DLQ due to handler errors", "event_type", env.Type, "msg_id", msg.ID)
		b.pushToDLQ(ctx, env.Type, msg.Values)
	}
}

// getHandlers retrieves a copy of the handlers for a given event type.
func (b *RedisEventBus) getHandlers(eventType string) []eventbus.HandlerFunc {
	b.handlersMtx.RLock()
	defer b.handlersMtx.RUnlock()

	handlers := make([]eventbus.HandlerFunc, len(b.handlers[eventType]))
	copy(handlers, b.handlers[eventType])
	return handlers
}

// executeHandlers runs all handlers for an event and returns true if all succeed.
func (b *RedisEventBus) executeHandlers(ctx context.Context, eventType string, evt events.Event, msgID string, handlers []eventbus.HandlerFunc) bool {
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := true

	for _, handler := range handlers {
		wg.Add(1)
		go func(h eventbus.HandlerFunc) {
			defer wg.Done()
			if err := h(ctx, evt); err != nil {
				mu.Lock()
				success = false
				mu.Unlock()
				b.logger.Error("handler error", "error", err, "event_type", eventType, "msg_id", msgID)
			}
		}(handler)
	}

	wg.Wait()
	return success
}

// ackMessage acknowledges a message in the Redis stream.
func (b *RedisEventBus) ackMessage(ctx context.Context, eventType, group, msgID string) error {
	stream := b.streamFor(eventType)
	return b.client.XAck(ctx, stream, group, msgID).Err()
}

// pushToDLQ pushes the raw event (msg.Values) to a DLQ Redis stream for inspection or reprocessing.
func (b *RedisEventBus) pushToDLQ(ctx context.Context, eventType string, values map[string]any) {
	dlqStream := b.dlqStreamName(eventType)
	err := b.publishToStream(ctx, dlqStream, values)
	b.logDLQResult(err, dlqStream, values)
}

// dlqStreamName returns the DLQ stream name for the given event type.
func (b *RedisEventBus) dlqStreamName(eventType string) string {
	return fmt.Sprintf("%s-%s-DLQ", b.streamFor(eventType), eventType)
}

// publishToStream adds a raw message to a given Redis stream.
func (b *RedisEventBus) publishToStream(ctx context.Context, stream string, values map[string]any) error {
	_, err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	return err
}

// logDLQResult logs the result of a DLQ operation.
func (b *RedisEventBus) logDLQResult(err error, stream string, values map[string]any) {
	if err != nil {
		b.logger.Error("failed to push to DLQ", "error", err, "stream", stream)
	} else {
		b.logger.Warn("event pushed to DLQ", "stream", stream, "values", values)
	}
}
