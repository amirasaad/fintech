//go:build redis
// +build redis

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

// RedisEventBus implements a production-ready event bus using Redis Streams.
// RedisEventBusConfig holds configuration for the Redis event bus
type RedisEventBusConfig struct {
	// DLQRetryInterval specifies how often to retry DLQ messages
	DLQRetryInterval time.Duration
	// DLQBatchSize specifies how many messages to process in each batch
	DLQBatchSize int64
	// DLQMaxRetries specifies the maximum number of retries per message before giving up
	DLQMaxRetries int
	// DLQInitialBackoff specifies the initial backoff duration for exponential backoff
	DLQInitialBackoff time.Duration
	// DLQMaxBackoff specifies the maximum backoff duration
	DLQMaxBackoff time.Duration
}

// DefaultRedisEventBusConfig returns the default configuration for RedisEventBus
func DefaultRedisEventBusConfig() *RedisEventBusConfig {
	return &RedisEventBusConfig{
		DLQRetryInterval:  5 * time.Minute,  // Check DLQ every 5 minutes
		DLQBatchSize:      10,               // Process 10 messages per batch
		DLQMaxRetries:     5,                // Maximum 5 retries per message
		DLQInitialBackoff: 1 * time.Minute,  // Start with 1 minute backoff
		DLQMaxBackoff:     30 * time.Minute, // Cap at 30 minutes
	}
}

type RedisEventBus struct {
	client      *redis.Client
	handlers    map[events.EventType][]eventbus.HandlerFunc
	handlersMtx sync.RWMutex
	dlqMtx      sync.Mutex // Protects DLQ-related fields
	logger      *slog.Logger
	config      *RedisEventBusConfig
	cancelFunc  context.CancelFunc
	wg          sync.WaitGroup
	dlqStopChan chan struct{}
	dlqStopped  chan struct{}
}

// NewWithRedis creates a new Redis-backed event bus.
// url: Redis connection URL (e.g., "redis://localhost:6379")
// config: Optional configuration. If nil, default values will be used.
func NewWithRedis(
	url string,
	logger *slog.Logger,
	config *RedisEventBusConfig,
) (*RedisEventBus, error) {
	if url == "" {
		return nil, fmt.Errorf("redis event bus: url is required")
	}

	// Use default config if none provided
	if config == nil {
		config = DefaultRedisEventBusConfig()
	}

	client, err := setupRedisClient(url)
	if err != nil {
		return nil, err
	}

	// Initialize logger if nil
	if logger == nil {
		logger = slog.Default()
	}

	bus := createRedisEventBus(client, logger, config)

	// Start background DLQ retry worker with a background context that's not cancelled
	// when the parent context is cancelled
	dlqCtx, cancel := context.WithCancel(context.Background())
	if err := bus.startDLQRetryWorker(dlqCtx); err != nil {
		cancel()
		bus.logger.Error("❌ Failed to start DLQ retry worker", "error", err)
		return nil, fmt.Errorf("failed to start DLQ retry worker: %w", err)
	}
	// Store the cancel function to stop the worker when the bus is closed
	bus.cancelFunc = cancel

	// Log successful initialization
	logger.Info("🚀 Redis event bus initialized with DLQ retry worker",
		"dlq_retry_interval", config.DLQRetryInterval,
		"dlq_batch_size", config.DLQBatchSize,
	)

	return bus, nil
}

// Emit publishes an event to the Redis stream.
func (b *RedisEventBus) Emit(
	ctx context.Context,
	event events.Event,
) error {
	b.logger.Debug(" Emitting event",
		"event_type", event.Type(),
		"event", fmt.Sprintf("%+v", event),
	)
	if err := b.validateClient(); err != nil {
		return err
	}
	envBytes, err := b.buildEnvelope(event)
	if err != nil {
		return err
	}
	// Convert the event type string to EventType
	eventType := events.EventType(event.Type())
	if err := b.publishEnvelope(ctx, eventType, envBytes); err != nil {
		return err
	}
	b.logger.Debug(
		"event emitted successfully",
		"type", eventType,
	)
	return nil
}

// Register registers an event handler for a specific event type.
func (b *RedisEventBus) Register(
	eventType events.EventType,
	handler eventbus.HandlerFunc,
) {
	b.logger.Debug(
		"registering handler",
		"event_type", eventType,
	)
	b.logger.Debug("registering handler", "event_type", eventType)
	ctx := context.Background()
	b.registerHandler(eventType, handler)
	if err := b.startConsumerForEvent(ctx, eventType); err != nil {
		if !errors.Is(err, redis.Nil) {
			b.logger.Error(
				"error reading from stream",
				"error", err,
				"event_type", eventType,
			)
		}
	}
}

// createRedisEventBus initializes the RedisEventBus struct.
func createRedisEventBus(
	client *redis.Client,
	logger *slog.Logger,
	config *RedisEventBusConfig,
) *RedisEventBus {
	return &RedisEventBus{
		client:   client,
		handlers: make(map[events.EventType][]eventbus.HandlerFunc),
		logger:   logger.With("bus", "redis"),
		config:   config,
		// channels will be initialized when the DLQ worker actually starts
		dlqStopChan: nil,
		dlqStopped:  nil,
	}
}

// initializeConsumerGroup ensures group exists and cleans up idle consumers.
func (b *RedisEventBus) initializeConsumerGroup(
	ctx context.Context,
	eventType events.EventType,
) error {
	group := groupNameFor(eventType)
	stream := streamNameFor(eventType)
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

// startConsumerForEvent derives group/consumer names and starts consuming for
// eventType.
func (b *RedisEventBus) startConsumerForEvent(
	ctx context.Context,
	eventType events.EventType,
) error {
	if err := b.initializeConsumerGroup(ctx, eventType); err != nil {
		return err
	}
	b.startConsuming(ctx, eventType)
	return nil
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
func (b *RedisEventBus) ensureConsumerGroup(
	ctx context.Context,
	stream,
	group string,
) error {
	err := b.client.XGroupCreateMkStream(
		ctx,
		stream,
		group,
		"0", // start from the beginning so existing messages are consumable
	).Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// cleanupIdleConsumers removes idle consumers with pending messages.
func (b *RedisEventBus) cleanupIdleConsumers(
	ctx context.Context,
	stream,
	group string,
) {
	if consumers, err := b.client.XInfoConsumers(
		ctx,
		stream,
		group,
	).Result(); err == nil {
		for _, consumer := range consumers {
			idleDuration := consumer.Idle * time.Millisecond
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
		b.logger.Error(
			"failed to marshal event",
			"error", err,
			"event_type", event.Type(),
		)
		return nil, fmt.Errorf("redis event bus: marshal failed: %w", err)
	}

	env := envelope{Type: event.Type(), Payload: data}
	envBytes, err := json.Marshal(env)
	if err != nil {
		b.logger.Error(
			"failed to marshal envelope",
			"error", err,
			"type", event.Type(),
		)
		return nil, fmt.Errorf("redis event bus: envelope marshal failed: %w", err)
	}

	return envBytes, nil
}

// publishEnvelope adds the envelope to the redis stream.
func (b *RedisEventBus) publishEnvelope(
	ctx context.Context,
	eventType events.EventType,
	envelopeData []byte,
) error {
	args := b.prepareAddArgs(eventType, envelopeData)
	_, err := b.client.XAdd(ctx, args).Result()
	if err != nil {
		b.logger.Error(
			"failed to publish event",
			"error", err,
			"event_type", eventType,
		)
		return fmt.Errorf("failed to publish event: %w", err)
	}
	b.logger.Debug(
		"event published to stream",
		"event_type", eventType,
		"stream", args.Stream,
	)
	return nil
}

// prepareAddArgs prepares the XAddArgs for publishing an envelope.
func (b *RedisEventBus) prepareAddArgs(eventType events.EventType, data []byte) *redis.XAddArgs {
	stream := streamNameFor(eventType)
	return &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{"event": string(data)},
	}
}

// registerHandler safely registers a handler for the given event type.
func (b *RedisEventBus) registerHandler(eventType events.EventType, handler eventbus.HandlerFunc) {
	b.handlersMtx.Lock()
	defer b.handlersMtx.Unlock()
	b.ensureHandlersMap()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.logger.Debug("registered handler", "event_type", eventType)
}

// ensureHandlersMap initializes the handlers map if it is nil.
func (b *RedisEventBus) ensureHandlersMap() {
	if b.handlers == nil {
		b.handlers = make(map[events.EventType][]eventbus.HandlerFunc)
	}
}

// startConsuming starts a goroutine to consume events for the given eventType.
func (b *RedisEventBus) startConsuming(ctx context.Context, eventType events.EventType) {
	go b.consume(ctx, eventType)
}

// consume starts consuming messages from the
// Redis stream and routes them to the appropriate handlers.
func (b *RedisEventBus) consume(
	ctx context.Context,
	eventType events.EventType,
) {
	stream := streamNameFor(eventType)
	group := groupNameFor(eventType)
	consumer := consumerNameFor(eventType)
	b.logger.Debug(
		"starting consumer",
		"event_type", eventType,
		"stream", stream,
		"group", group,
		"consumer", consumer,
	)

	for {
		// Read messages from the stream
		messages, err := b.readStream(ctx, stream, group, consumer)
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				b.logger.Error(
					"error reading from stream",
					"error", err,
					"stream", stream,
					"group", group,
				)
			}
			time.Sleep(5 * time.Second) // Prevent tight loop on errors
			continue
		}

		// Process each message
		for _, msg := range messages {
			b.processMessage(ctx, group, msg)
		}
	}
}

// readStream reads messages from a redis stream group.
func (b *RedisEventBus) readStream(
	ctx context.Context,
	stream, group, consumer string,
) ([]redis.XMessage, error) {
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
			b.logger.Error(
				"error reading from stream",
				"error", err,
				"group", group,
			)
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
func (b *RedisEventBus) processMessage(
	ctx context.Context,
	group string,
	msg redis.XMessage,
) {
	b.logger.Debug("📥 Processing message",
		"msg_id", msg.ID,
		"group", group,
		"values", msg.Values,
	)
	raw, ok := msg.Values["event"].(string)
	if !ok {
		b.logger.Error(
			"invalid message format, missing 'event' field",
			"msg_id", msg.ID,
		)
		return
	}

	var env envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		b.logger.Error(
			"failed to unmarshal envelope",
			"error", err,
			"msg_id", msg.ID,
		)
		return
	}

	// Convert the string event type to EventType
	evtType := events.EventType(env.Type)

	constructor, ok := events.EventTypes[evtType]
	if !ok {
		b.logger.Error(
			"unknown event type",
			"type", env.Type,
			"msg_id", msg.ID,
		)
		_ = b.ackMessage(ctx, evtType, group, msg.ID)
		return
	}

	evt := constructor()

	b.logger.Debug("🔍 Unmarshaling event",
		"event_type", env.Type,
		"payload", string(env.Payload),
	)

	// Special handling for events with custom JSON unmarshaling
	err := json.Unmarshal(env.Payload, evt)

	b.logger.Debug("🔍 Unmarshaled event",
		"event_type", env.Type,
		"event", fmt.Sprintf("%+v", evt),
		"error", err,
	)

	if err != nil {
		b.logger.Error(
			"failed to unmarshal event",
			"error", err,
			"event_type", env.Type,
			"msg_id", msg.ID,
		)
		_ = b.ackMessage(ctx, evtType, group, msg.ID)
		return
	}

	handlers := b.getHandlers(evtType)
	if len(handlers) == 0 {
		b.logger.Warn(
			"no handlers registered for event type",
			"event_type", env.Type,
			"msg_id", msg.ID,
		)
		_ = b.ackMessage(ctx, evtType, group, msg.ID)
		return
	}

	success := b.executeHandlers(
		ctx,
		evtType,
		evt,
		msg.ID,
		handlers,
	)

	if success {
		if err := b.ackMessage(ctx, evtType, group, msg.ID); err != nil {
			b.logger.Error(
				"failed to ack message",
				"error", err,
				"msg_id", msg.ID,
			)
		}
	} else {
		b.logger.Warn(
			"sending message to DLQ due to handler errors",
			"event_type", env.Type,
			"msg_id", msg.ID,
		)
		b.pushToDLQ(ctx, evtType, msg.Values)
		// Ack the original message to avoid reprocessing duplicates endlessly
		if err := b.ackMessage(ctx, evtType, group, msg.ID); err != nil {
			b.logger.Error(
				"failed to ack message after DLQ push",
				"error", err,
				"msg_id", msg.ID,
			)
		}
	}
}

// getHandlers retrieves a copy of the handlers for a given event type.
func (b *RedisEventBus) getHandlers(
	eventType events.EventType,
) []eventbus.HandlerFunc {
	b.handlersMtx.RLock()
	defer b.handlersMtx.RUnlock()

	handlers, exists := b.handlers[eventType]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	handlersCopy := make([]eventbus.HandlerFunc, len(handlers))
	copy(handlersCopy, handlers)
	return handlersCopy
}

// executeHandlers runs all handlers for an event and returns true if all succeed.
func (b *RedisEventBus) executeHandlers(
	ctx context.Context,
	eventType events.EventType,
	evt events.Event,
	msgID string,
	handlers []eventbus.HandlerFunc,
) bool {
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
				b.logger.Error(
					"handler error",
					"error", err,
					"event_type", eventType,
					"msg_id", msgID,
				)
			}
		}(handler)
	}

	wg.Wait()
	return success
}

// ackMessage acknowledges a message in the Redis stream.
func (b *RedisEventBus) ackMessage(
	ctx context.Context,
	eventType events.EventType,
	group, msgID string,
) error {
	stream := streamNameFor(eventType)
	_, err := b.client.XAck(ctx, stream, group, msgID).Result()
	return err
}

// pushToDLQ pushes the raw event (msg.Values) to a DLQ Redis stream for inspection or reprocessing.
func (b *RedisEventBus) pushToDLQ(
	ctx context.Context,
	eventType events.EventType,
	values map[string]any,
) {
	dlqStream := dlqStreamName(eventType)
	b.logger.Info("pushing message to DLQ",
		"event_type", eventType,
		"dlq_stream", dlqStream,
	)
	if err := b.publishToStream(ctx, dlqStream, values); err != nil {
		b.logDLQResult(err, dlqStream, values)
	} else {
		b.logger.Info("successfully pushed message to DLQ",
			"event_type", eventType,
			"dlq_stream", dlqStream,
		)
	}
}

// publishToStream adds a raw message to a given Redis stream.
func (b *RedisEventBus) publishToStream(
	ctx context.Context,
	stream string,
	values map[string]any,
) error {
	_, err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	return err
}

// logDLQResult logs the result of a DLQ operation.
func (b *RedisEventBus) logDLQResult(
	err error,
	stream string,
	values map[string]any,
) {
	if err != nil {
		b.logger.Error(
			"failed to push to DLQ",
			"error", err,
			"stream", stream,
		)
	} else {
		b.logger.Warn(
			"event pushed to DLQ",
			"stream", stream,
			"values", values,
		)
	}
}

// startDLQRetryWorker starts a background worker that periodically processes DLQ messages
// for all event types. The worker will run until
// the context is cancelled or StopDLQRetryWorker is called.
//
// This method is idempotent - calling it multiple times will
// have no effect if the worker is already running.
// If the worker is not running but channels exist from a previous run, they will be cleaned up.
// startDLQRetryWorker starts a background worker that processes DLQ messages.
// It's safe to call this method multiple times - it will only start one worker.
// The worker will run until the context is cancelled or StopDLQRetryWorker is called.
func (b *RedisEventBus) startDLQRetryWorker(ctx context.Context) error {
	b.dlqMtx.Lock()
	defer b.dlqMtx.Unlock()

	// Add a debug log to track when this function is called
	b.logger.Debug("startDLQRetryWorker called",
		"dlq_retry_interval", b.config.DLQRetryInterval,
	)

	// If worker is already running, return nil to make this call idempotent
	if b.dlqStopChan != nil {
		// Check if the worker is actually running by checking if the stopped channel is closed
		select {
		case <-b.dlqStopped:
			// Worker has stopped, clean up and continue to restart
			b.logger.Info("previous DLQ worker has stopped, cleaning up and restarting")
			// Reset channels to allow restart
			b.dlqStopChan = nil
			b.dlqStopped = nil
		default:
			// Worker is still running, no action needed
			b.logger.Info("DLQ retry worker is already running",
				"dlq_retry_interval", b.config.DLQRetryInterval,
				"dlq_batch_size", b.config.DLQBatchSize,
			)
			return nil
		}
	}

	// Create new channels for this worker instance
	b.dlqStopChan = make(chan struct{})
	b.dlqStopped = make(chan struct{})

	// Create a new context that will be cancelled when the worker is stopped
	var cancelCtx context.Context
	cancelCtx, b.cancelFunc = context.WithCancel(ctx)

	// Log the worker startup with configuration details
	b.logger.Info("Starting DLQ retry worker",
		"dlq_retry_interval", b.config.DLQRetryInterval,
		"dlq_batch_size", b.config.DLQBatchSize,
		"num_event_types", len(events.EventTypes),
	)

	b.wg.Add(1)
	go func(ctx context.Context, logger *slog.Logger) {
		logger.Info("Starting DLQ retry worker",
			"interval", b.config.DLQRetryInterval,
			"batch_size", b.config.DLQBatchSize)

		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in DLQ worker: %v", r)
				logger.Error(
					"DLQ retry worker panicked",
					"error", err)
			}
			// Always signal stopped and mark WaitGroup done
			close(b.dlqStopped)
			b.wg.Done()
			// Ensure we don't leave any pending messages when shutting down
			if ctx.Err() == nil {
				b.processAllDLQs(context.Background())
			}
			logger.Info("DLQ retry worker stopped")
		}()

		ticker := time.NewTicker(b.config.DLQRetryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("stopping DLQ retry worker: context cancelled")
				return
			case <-b.dlqStopChan:
				logger.Info("stopping DLQ retry worker: stop requested")
				return
			case <-ticker.C:
				logger.Debug("DLQ retry worker tick - processing DLQs")
				start := time.Now()
				b.processAllDLQs(ctx)
				logger.Debug("DLQ processing completed", "duration", time.Since(start))
			}
		}
	}(cancelCtx, b.logger)

	// Start a goroutine to clean up when the worker stops
	go func() {
		<-b.dlqStopped
		b.dlqMtx.Lock()
		defer b.dlqMtx.Unlock()

		// Only clean up if the channels still point to the ones we created
		if b.dlqStopChan != nil && b.dlqStopped != nil {
			b.dlqStopChan = nil
			b.dlqStopped = nil
		}
	}()

	return nil
}

// processAllDLQs processes DLQ messages for all registered event types
func (b *RedisEventBus) processAllDLQs(ctx context.Context) {
	if ctx.Err() != nil {
		b.logger.Debug("skipping DLQ processing: context cancelled")
		return
	}

	b.logger.Debug("processing DLQs for all event types",
		"num_event_types", len(events.EventTypes),
		"batch_size", b.config.DLQBatchSize,
	)

	// Ensure DLQ consumer group exists for each event type
	for eventType := range events.EventTypes {
		dlq := dlqStreamName(eventType)
		b.logger.Debug("ensuring consumer group for DLQ",
			"event_type", eventType,
			"dlq_stream", dlq,
		)
		if err := b.ensureConsumerGroup(ctx, dlq, "dlq-retry-worker"); err != nil {
			b.logger.Error("failed to ensure DLQ consumer group",
				"error", err,
				"dlq_stream", dlq,
				"group", "dlq-retry-worker",
			)
		} else {
			b.logger.Debug("successfully ensured consumer group for DLQ",
				"event_type", eventType,
				"dlq_stream", dlq,
			)
		}
	}

	processedAny := false
	for eventType := range events.EventTypes {
		dlq := dlqStreamName(eventType)

		// Check if DLQ exists and has messages before processing
		exists, err := b.client.Exists(ctx, dlq).Result()
		if err != nil {
			b.logger.Error("failed to check DLQ existence",
				"error", err,
				"dlq_stream", dlq,
			)
			continue
		}

		if exists == 0 {
			// DLQ doesn't exist for this event type, skip
			b.logger.Debug("DLQ does not exist, skipping",
				"event_type", eventType,
				"dlq_stream", dlq,
			)
			continue
		}

		// Check if there are any messages in the DLQ
		streamLen, err := b.client.XLen(ctx, dlq).Result()
		if err != nil {
			b.logger.Error("failed to get DLQ length",
				"error", err,
				"dlq_stream", dlq,
			)
			continue
		}

		if streamLen == 0 {
			b.logger.Debug("📭 DLQ is empty, skipping",
				"event_type", eventType,
				"dlq_stream", dlq,
			)
			continue
		}

		stream := streamNameFor(eventType)
		b.logger.Info("🔄 Processing DLQ messages",
			"event_type", eventType,
			"dlq_stream", dlq,
			"target_stream", stream,
			"message_count", streamLen,
		)

		if err := b.retryDLQ(ctx, dlq, stream, b.config.DLQBatchSize); err != nil {
			b.logger.Error("❌ Failed to process DLQ messages",
				"event_type", eventType,
				"error", err,
			)
		} else {
			processedAny = true
			b.logger.Info("✅ Successfully processed DLQ messages",
				"event_type", eventType,
				"dlq_stream", dlq,
				"message_count", streamLen,
			)
		}
	}

	if !processedAny {
		b.logger.Debug("📭 No DLQ messages to process")
	}
}

// retryDLQ reads messages from the DLQ and republishes them to the original stream
func (b *RedisEventBus) retryDLQ(
	ctx context.Context,
	dlqStream,
	originalStream string,
	count int64,
) error {
	// Add a timeout to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	b.logger.Debug("attempting to retry DLQ messages",
		"dlq_stream", dlqStream,
		"target_stream", originalStream,
		"batch_size", count,
	)

	// First check if the DLQ stream exists
	exists, err := b.client.Exists(ctx, dlqStream).Result()
	if err != nil {
		b.logger.Error("failed to check DLQ existence",
			"error", err,
			"dlq_stream", dlqStream,
		)
		return fmt.Errorf("failed to check DLQ existence: %w", err)
	}
	if exists == 0 {
		b.logger.Debug("DLQ stream does not exist, nothing to process",
			"dlq_stream", dlqStream,
		)
		return nil
	}

	// Get stream length for debugging
	streamLen, err := b.client.XLen(ctx, dlqStream).Result()
	if err != nil {
		b.logger.Warn("failed to get DLQ stream length",
			"error", err,
			"dlq_stream", dlqStream,
		)
	} else {
		b.logger.Debug("DLQ stream status",
			"dlq_stream", dlqStream,
			"message_count", streamLen,
		)
	}

	// Read messages from DLQ with a smaller batch size if count is too large
	if count <= 0 || count > 100 {
		count = 10 // Default to a reasonable batch size
	}

	b.logger.Debug("reading messages from DLQ",
		"dlq_stream", dlqStream,
		"batch_size", count,
	)

	// Ensure consumer group exists before reading
	if err := b.ensureConsumerGroup(ctx, dlqStream, "dlq-retry-worker"); err != nil {
		b.logger.Warn("failed to ensure DLQ consumer group, falling back to direct read",
			"error", err,
			"dlq_stream", dlqStream,
		)
	}

	var entries []redis.XMessage
	var readViaGroup bool // Track if we read via consumer group (needed for ACK)

	// Use XReadGroup to read from DLQ to prevent message loss
	streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    "dlq-retry-worker",
		Consumer: "dlq-consumer",
		Streams:  []string{dlqStream, ">"},
		Count:    count,
		Block:    100 * time.Millisecond,
	}).Result()

	switch {
	case err == nil && len(streams) > 0:
		entries = streams[0].Messages
		readViaGroup = true
	case errors.Is(err, redis.Nil):
		// No new messages for the consumer group, try reading pending messages
		pending, pendErr := b.client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: dlqStream,
			Group:  "dlq-retry-worker",
			Start:  "-",
			End:    "+",
			Count:  count,
		}).Result()
		if pendErr == nil && len(pending) > 0 {
			// Read pending messages
			ids := make([]string, len(pending))
			for i, p := range pending {
				ids[i] = p.ID
			}
			pendingMsgs, pendReadErr := b.client.XClaim(ctx, &redis.XClaimArgs{
				Stream:   dlqStream,
				Group:    "dlq-retry-worker",
				Consumer: "dlq-consumer",
				MinIdle:  0,
				Messages: ids,
			}).Result()
			if pendReadErr == nil && len(pendingMsgs) > 0 {
				entries = pendingMsgs
				readViaGroup = true
				b.logger.Debug("claimed pending messages from DLQ",
					"count", len(entries),
					"dlq_stream", dlqStream,
				)
			}
		}

		// If still no entries, fall back to XREAD from start
		// (for messages that existed before group creation)
		if len(entries) == 0 {
			b.logger.Debug("no group messages; attempting backlog read from start",
				"dlq_stream", dlqStream,
				"batch_size", count,
			)
			raw, rerr := b.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{dlqStream, "0"},
				Count:   count,
				Block:   100 * time.Millisecond,
			}).Result()
			if rerr != nil && !errors.Is(rerr, redis.Nil) {
				b.logger.Error("failed backlog read from DLQ",
					"error", rerr,
					"dlq_stream", dlqStream,
				)
				return fmt.Errorf("failed backlog read from DLQ: %w", rerr)
			}
			if len(raw) > 0 && len(raw[0].Messages) > 0 {
				entries = raw[0].Messages
			}
		}
	case err != nil:
		// Only log non-Nil errors as warnings
		b.logger.Warn("XReadGroup returned error (will try fallback)",
			"error", err,
			"dlq_stream", dlqStream,
		)
		// Try XREAD as fallback for other errors
		if len(entries) == 0 {
			raw, rerr := b.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{dlqStream, "0"},
				Count:   count,
				Block:   100 * time.Millisecond,
			}).Result()
			if rerr != nil && !errors.Is(rerr, redis.Nil) {
				return fmt.Errorf("failed fallback read from DLQ: %w", rerr)
			}
			if len(raw) > 0 && len(raw[0].Messages) > 0 {
				entries = raw[0].Messages
			}
		}
	}

	if len(entries) == 0 {
		b.logger.Debug("no messages found in DLQ",
			"dlq_stream", dlqStream,
		)
		return nil
	}

	b.logger.Info("found messages in DLQ",
		"count", len(entries),
		"dlq_stream", dlqStream,
	)

	var retryCount int
	var skippedCount int
	var lastErr error

	// Process each message
	for _, entry := range entries {
		data, ok := entry.Values["event"]
		if !ok || data == nil {
			b.logger.Warn("DLQ message missing event data", "message_id", entry.ID)
			continue
		}

		// Get retry count from message metadata (stored as "retry_count")
		retryAttempt := 0
		if retryCountStr, ok := entry.Values["retry_count"].(string); ok {
			parsed, err := fmt.Sscanf(retryCountStr, "%d", &retryAttempt)
			if err != nil || parsed != 1 {
				// If parsing fails, retryAttempt remains 0
				retryAttempt = 0
			}
		}

		// Check if message has exceeded max retries
		if retryAttempt >= b.config.DLQMaxRetries {
			b.logger.Warn("⚠️ Message exceeded max retries, skipping",
				"message_id", entry.ID,
				"retry_count", retryAttempt,
				"max_retries", b.config.DLQMaxRetries,
				"dlq_stream", dlqStream,
			)
			skippedCount++
			// Optionally move to a permanent failure queue or delete
			if _, err := b.client.XDel(ctx, dlqStream, entry.ID).Result(); err != nil {
				b.logger.Warn("Failed to delete exhausted message from DLQ",
					"error", err,
					"message_id", entry.ID,
				)
			}
			continue
		}

		// Calculate exponential backoff delay
		backoffDuration := b.calculateBackoff(retryAttempt)
		if backoffDuration > 0 {
			b.logger.Debug("Applying exponential backoff before retry",
				"message_id", entry.ID,
				"retry_attempt", retryAttempt,
				"backoff_duration", backoffDuration,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffDuration):
				// Continue with retry after backoff
			}
		}

		// Increment retry count for next attempt
		newRetryCount := retryAttempt + 1

		// Republish to original stream with updated retry count
		if _, err := b.client.XAdd(ctx, &redis.XAddArgs{
			Stream: originalStream,
			Values: map[string]any{
				"event":       data,
				"retry_count": fmt.Sprintf("%d", newRetryCount),
			},
		}).Result(); err != nil {
			lastErr = fmt.Errorf("failed to republish message %s: %w", entry.ID, err)
			b.logger.Error("Failed to republish DLQ message",
				"error", err,
				"message_id", entry.ID,
				"retry_attempt", retryAttempt,
				"dlq_stream", dlqStream,
				"target_stream", originalStream,
			)
			continue
		}

		// Acknowledge the message in the DLQ after successful republish
		// Only ACK if we read via consumer group (XReadGroup or XClaim)
		// Messages read via XRead don't need ACK
		if readViaGroup {
			if _, err := b.client.XAck(
				ctx,
				dlqStream,
				"dlq-retry-worker",
				entry.ID,
			).Result(); err != nil {
				b.logger.Error("Failed to acknowledge message in DLQ",
					"error", err,
					"message_id", entry.ID,
					"dlq_stream", dlqStream,
				)
			}
		}

		// Also try to delete the message to prevent DLQ from growing
		if _, err := b.client.XDel(
			ctx,
			dlqStream,
			entry.ID,
		).Result(); err != nil {
			// Log but don't fail the entire batch if delete fails
			b.logger.Warn("Failed to delete retried message from DLQ",
				"error", err,
				"message_id", entry.ID,
				"dlq_stream", dlqStream,
			)
		}

		retryCount++
		b.logger.Info("✅ Retried DLQ message",
			"message_id", entry.ID,
			"retry_attempt", newRetryCount,
			"dlq_stream", dlqStream,
		)
	}

	if retryCount > 0 {
		b.logger.Info("✅ Successfully retried DLQ messages",
			"count", retryCount,
			"skipped", skippedCount,
			"dlq_stream", dlqStream,
			"target_stream", originalStream,
		)
	}
	if skippedCount > 0 {
		b.logger.Warn("⚠️ Skipped messages that exceeded max retries",
			"count", skippedCount,
			"max_retries", b.config.DLQMaxRetries,
		)
	}

	// Return the last error if all retries failed
	if retryCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to retry any messages: %w", lastErr)
	}

	return nil
}

// calculateBackoff calculates the exponential backoff duration for a given retry attempt.
// It uses the formula: min(initialBackoff * 2^attempt, maxBackoff)
func (b *RedisEventBus) calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	if attempt < 0 {
		attempt = 0
	}

	// Calculate exponential backoff: initial * 2^attempt
	backoff := b.config.DLQInitialBackoff
	for i := 0; i < attempt; i++ {
		backoff *= 2
		// Cap at max backoff
		if backoff > b.config.DLQMaxBackoff {
			backoff = b.config.DLQMaxBackoff
			break
		}
	}

	// Ensure we don't exceed max backoff
	if backoff > b.config.DLQMaxBackoff {
		backoff = b.config.DLQMaxBackoff
	}

	return backoff
}
