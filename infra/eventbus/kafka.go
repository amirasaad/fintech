//go:build kafka
// +build kafka

package eventbus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// KafkaEventBusConfig holds configuration for the Kafka event bus.
type KafkaEventBusConfig struct {
	GroupID          string
	TopicPrefix      string
	DLQRetryInterval time.Duration
	DLQBatchSize     int
	SASLUsername     string
	SASLPassword     string
	TLSEnabled       bool
	TLSCAFile        string
	TLSCertFile      string
	TLSKeyFile       string
	TLSSkipVerify    bool
}

// DefaultKafkaEventBusConfig returns default configuration for KafkaEventBus.
func DefaultKafkaEventBusConfig() *KafkaEventBusConfig {
	return &KafkaEventBusConfig{
		GroupID:          "fintech",
		TopicPrefix:      "fintech.events",
		DLQRetryInterval: 5 * time.Minute,
		DLQBatchSize:     10,
	}
}

// KafkaEventBus implements a Kafka-backed event bus.
type KafkaEventBus struct {
	brokers []string
	writer  *kafka.Writer
	dialer  *kafka.Dialer
	ctx     context.Context

	handlers    map[events.EventType][]eventbus.HandlerFunc
	handlersMtx sync.RWMutex

	readers    map[events.EventType]*kafka.Reader
	readersMtx sync.Mutex
	topicsMtx  sync.Mutex
	topics     map[string]struct{}

	logger *slog.Logger
	config *KafkaEventBusConfig

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWithKafka creates a new Kafka-backed event bus.
// brokers: Comma-separated brokers list (e.g. "localhost:9092,localhost:9093").
func NewWithKafka(
	brokers string,
	logger *slog.Logger,
	config *KafkaEventBusConfig,
) (*KafkaEventBus, error) {
	parsedBrokers := parseBrokers(brokers)
	if len(parsedBrokers) == 0 {
		return nil, fmt.Errorf("kafka event bus: brokers are required")
	}

	if config == nil {
		config = DefaultKafkaEventBusConfig()
	}
	if config.GroupID == "" {
		config.GroupID = "fintech"
	}
	if strings.TrimSpace(config.TopicPrefix) == "" {
		config.TopicPrefix = "fintech.events"
	}
	if config.DLQBatchSize <= 0 {
		config.DLQBatchSize = 10
	}
	if config.DLQRetryInterval <= 0 {
		config.DLQRetryInterval = 5 * time.Minute
	}

	if logger == nil {
		logger = slog.Default()
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(parsedBrokers...),
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		Balancer:               &kafka.Hash{},
	}

	dialer, transport, err := newKafkaDialer(config)
	if err != nil {
		return nil, err
	}
	writer.Dialer = dialer
	if transport != nil {
		writer.Transport = transport
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &KafkaEventBus{
		brokers:  parsedBrokers,
		writer:   writer,
		dialer:   dialer,
		ctx:      ctx,
		handlers: make(map[events.EventType][]eventbus.HandlerFunc),
		readers:  make(map[events.EventType]*kafka.Reader),
		topics:   make(map[string]struct{}),
		logger:   logger.With("bus", "kafka"),
		config:   config,
		cancel:   cancel,
	}

	if err := bus.ping(ctx); err != nil {
		_ = bus.Close()
		return nil, err
	}

	bus.startDLQRetryWorker(ctx)
	logger.Info("🚀 Kafka event bus initialized",
		"group_id", config.GroupID,
		"brokers", parsedBrokers,
		"dlq_retry_interval", config.DLQRetryInterval,
		"dlq_batch_size", config.DLQBatchSize,
		"tls_enabled", dialer.TLS != nil,
		"sasl_enabled", dialer.SASLMechanism != nil,
	)

	return bus, nil
}

// Close stops background goroutines and closes network resources.
func (b *KafkaEventBus) Close() error {
	if b == nil {
		return nil
	}

	if b.cancel != nil {
		b.cancel()
	}

	b.readersMtx.Lock()
	for _, r := range b.readers {
		_ = r.Close()
	}
	b.readersMtx.Unlock()

	b.wg.Wait()

	if b.writer != nil {
		return b.writer.Close()
	}
	return nil
}

// Register registers an event handler for a specific event type.
func (b *KafkaEventBus) Register(eventType events.EventType, handler eventbus.HandlerFunc) {
	b.handlersMtx.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.handlersMtx.Unlock()

	b.ensureConsumer(eventType)
}

// Emit publishes an event to Kafka.
func (b *KafkaEventBus) Emit(ctx context.Context, event events.Event) error {
	if b == nil || b.writer == nil {
		return fmt.Errorf("kafka event bus: writer not initialized")
	}

	envBytes, err := b.buildEnvelope(event)
	if err != nil {
		return err
	}

	eventType := events.EventType(event.Type())
	topic := topicNameFor(b.config.TopicPrefix, eventType)
	if err := b.ensureTopic(ctx, topic); err != nil {
		return err
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(event.Type()),
		Value: envBytes,
		Time:  time.Now(),
	}

	if err := b.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka event bus: publish failed: %w", err)
	}
	return nil
}

func (b *KafkaEventBus) ping(ctx context.Context) error {
	conn, err := b.getDialer().DialContext(ctx, "tcp", b.brokers[0])
	if err != nil {
		return fmt.Errorf("kafka event bus: connection failed: %w", err)
	}
	_ = conn.Close()
	return nil
}

func (b *KafkaEventBus) ensureConsumer(eventType events.EventType) {
	b.readersMtx.Lock()
	defer b.readersMtx.Unlock()

	if _, exists := b.readers[eventType]; exists {
		return
	}

	if err := b.ensureTopic(b.ctx, topicNameFor(b.config.TopicPrefix, eventType)); err != nil {
		b.logger.Error("kafka ensure topic error", "error", err, "event_type", eventType)
		return
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     b.brokers,
		GroupID:     b.config.GroupID,
		Topic:       topicNameFor(b.config.TopicPrefix, eventType),
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     1 * time.Second,
		Dialer:      b.getDialer(),
	})
	b.readers[eventType] = reader

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.consumeLoop(b.ctx, eventType, reader)
	}()
}

func (b *KafkaEventBus) consumeLoop(ctx context.Context, eventType events.EventType, reader *kafka.Reader) {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errorsIsContextCanceled(err) {
				return
			}
			b.logger.Error("kafka consume error", "error", err, "topic", reader.Stats().Topic, "event_type", eventType)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		shouldCommit, commitErr := b.processKafkaMessage(ctx, eventType, msg)
		if shouldCommit {
			if err := reader.CommitMessages(ctx, msg); err != nil {
				b.logger.Error("kafka commit error", "error", err, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
			}
		} else if commitErr != nil {
			b.logger.Error("kafka message processing failed; will retry", "error", commitErr, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (b *KafkaEventBus) processKafkaMessage(
	ctx context.Context,
	expectedType events.EventType,
	msg kafka.Message,
) (commit bool, processingErr error) {
	var env envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		b.logger.Error("failed to unmarshal envelope", "error", err, "topic", msg.Topic, "offset", msg.Offset)
		return true, nil
	}

	evtType := events.EventType(env.Type)
	if evtType == "" {
		b.logger.Error("missing event type in envelope", "topic", msg.Topic, "offset", msg.Offset)
		return true, nil
	}
	if expectedType != "" && evtType != expectedType {
		b.logger.Warn("envelope type mismatch for topic", "expected", expectedType, "actual", evtType, "topic", msg.Topic)
	}

	constructor, ok := events.EventTypes[evtType]
	if !ok {
		b.logger.Error("unknown event type", "type", env.Type, "topic", msg.Topic, "offset", msg.Offset)
		return true, nil
	}

	evt := constructor()
	if err := json.Unmarshal(env.Payload, evt); err != nil {
		b.logger.Error("failed to unmarshal event payload", "error", err, "event_type", env.Type, "topic", msg.Topic, "offset", msg.Offset)
		return true, nil
	}

	handlers := b.getHandlers(evtType)
	if len(handlers) == 0 {
		b.logger.Warn("no handlers registered for event type", "event_type", evtType, "topic", msg.Topic, "offset", msg.Offset)
		return true, nil
	}

	success := executeHandlers(ctx, b.logger, evtType, evt, handlers, fmt.Sprintf("%d", msg.Offset))
	if success {
		return true, nil
	}

	if err := b.publishToDLQ(ctx, evtType, msg.Value); err != nil {
		return false, err
	}
	return true, nil
}

func (b *KafkaEventBus) publishToDLQ(ctx context.Context, eventType events.EventType, raw []byte) error {
	dlqTopic := dlqTopicNameFor(b.config.TopicPrefix, eventType)
	if err := b.ensureTopic(ctx, dlqTopic); err != nil {
		return err
	}
	msg := kafka.Message{
		Topic: dlqTopic,
		Key:   []byte(eventType.String()),
		Value: raw,
		Time:  time.Now(),
	}

	if err := b.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka event bus: dlq publish failed: %w", err)
	}
	b.logger.Warn("message sent to DLQ", "event_type", eventType, "dlq_topic", dlqTopic)
	return nil
}

func (b *KafkaEventBus) buildEnvelope(event events.Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("kafka event bus: marshal failed: %w", err)
	}
	env := envelope{Type: event.Type(), Payload: data}
	envBytes, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("kafka event bus: envelope marshal failed: %w", err)
	}
	return envBytes, nil
}

func (b *KafkaEventBus) getDialer() *kafka.Dialer {
	return b.dialer
}

func newKafkaDialer(config *KafkaEventBusConfig) (*kafka.Dialer, *kafka.Transport, error) {
	tlsConfig, err := buildKafkaTLSConfig(config)
	if err != nil {
		return nil, nil, err
	}
	saslMechanism, err := buildKafkaSASLMechanism(config)
	if err != nil {
		return nil, nil, err
	}

	dialer := &kafka.Dialer{
		Timeout:       5 * time.Second,
		TLS:           tlsConfig,
		SASLMechanism: saslMechanism,
	}

	if tlsConfig == nil && saslMechanism == nil {
		return dialer, nil, nil
	}

	transport := &kafka.Transport{
		TLS:  tlsConfig,
		SASL: saslMechanism,
	}
	return dialer, transport, nil
}

func buildKafkaTLSConfig(config *KafkaEventBusConfig) (*tls.Config, error) {
	// Only honor TLS when explicitly enabled
	if !config.TLSEnabled {
		return nil, nil

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: config.TLSSkipVerify,
	}

	caFile := strings.TrimSpace(config.TLSCAFile)
	if caFile != "" {
		caBytes, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("kafka event bus: read tls ca file: %w", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("kafka event bus: invalid tls ca file")
		}
		tlsConfig.RootCAs = caPool
	}

	certFile := strings.TrimSpace(config.TLSCertFile)
	keyFile := strings.TrimSpace(config.TLSKeyFile)
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			return nil, fmt.Errorf("kafka event bus: tls cert and key are required")
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("kafka event bus: load tls key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func buildKafkaSASLMechanism(config *KafkaEventBusConfig) (sasl.Mechanism, error) {
	username := strings.TrimSpace(config.SASLUsername)
	password := strings.TrimSpace(config.SASLPassword)
	if username == "" && password == "" {
		return nil, nil
	}
	if username == "" || password == "" {
		return nil, fmt.Errorf("kafka event bus: sasl username and password are required")
	}
	return plain.Mechanism{
		Username: username,
		Password: password,
	}, nil
}

func (b *KafkaEventBus) ensureTopic(ctx context.Context, topic string) error {
	if topic == "" {
		return fmt.Errorf("kafka event bus: topic is required")
	}

	b.topicsMtx.Lock()
	_, exists := b.topics[topic]
	b.topicsMtx.Unlock()
	if exists {
		return nil
	}

	conn, err := b.getDialer().DialContext(ctx, "tcp", b.brokers[0])
	if err != nil {
		return fmt.Errorf("kafka event bus: dial failed: %w", err)
	}
	defer func() { _ = conn.Close() }()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && !isTopicAlreadyExists(err) {
		return fmt.Errorf("kafka event bus: create topic failed: %w", err)
	}

	b.topicsMtx.Lock()
	b.topics[topic] = struct{}{}
	b.topicsMtx.Unlock()
	return nil
}

func isTopicAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Topic with this name already exists") ||
		strings.Contains(msg, "TOPIC_ALREADY_EXISTS") ||
		strings.Contains(msg, "TopicAlreadyExists")
}

func (b *KafkaEventBus) getHandlers(eventType events.EventType) []eventbus.HandlerFunc {
	b.handlersMtx.RLock()
	defer b.handlersMtx.RUnlock()
	handlers := b.handlers[eventType]
	out := make([]eventbus.HandlerFunc, len(handlers))
	copy(out, handlers)
	return out
}

func (b *KafkaEventBus) startDLQRetryWorker(ctx context.Context) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		ticker := time.NewTicker(b.config.DLQRetryInterval)
		defer ticker.Stop()

		b.processAllDLQs(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.processAllDLQs(ctx)
			}
		}
	}()
}

func (b *KafkaEventBus) processAllDLQs(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	eventTypes, err := b.listDLQEventTypes(ctx)
	if err != nil {
		b.logger.Error("failed to list DLQ topics", "error", err)
		return
	}
	for _, eventType := range eventTypes {
		b.retryDLQ(ctx, eventType, b.config.DLQBatchSize)
	}
}

func (b *KafkaEventBus) listDLQEventTypes(ctx context.Context) ([]events.EventType, error) {
	conn, err := b.getDialer().DialContext(ctx, "tcp", b.brokers[0])
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(partitions))
	out := make([]events.EventType, 0, len(partitions))
	prefix := strings.TrimSpace(b.config.TopicPrefix)
	if prefix == "" {
		prefix = "fintech.events"
	}
	dlqPrefix := fmt.Sprintf("%s.dlq.", prefix)
	for _, p := range partitions {
		if !strings.HasPrefix(p.Topic, dlqPrefix) {
			continue
		}
		eventTypeStr := strings.TrimPrefix(p.Topic, dlqPrefix)
		if eventTypeStr == "" {
			continue
		}
		if _, ok := seen[eventTypeStr]; ok {
			continue
		}
		seen[eventTypeStr] = struct{}{}
		out = append(out, events.EventType(eventTypeStr))
	}

	return out, nil
}

func (b *KafkaEventBus) retryDLQ(ctx context.Context, eventType events.EventType, batchSize int) {
	if batchSize <= 0 {
		batchSize = 10
	}

	dlqReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     b.brokers,
		GroupID:     b.config.GroupID + "-dlq-retry",
		Topic:       dlqTopicNameFor(b.config.TopicPrefix, eventType),
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     250 * time.Millisecond,
		Dialer:      b.getDialer(),
	})
	defer func() { _ = dlqReader.Close() }()

	for i := 0; i < batchSize; i++ {
		msgCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		msg, err := dlqReader.FetchMessage(msgCtx)
		cancel()
		if err != nil {
			return
		}

		env := envelope{}
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			_ = dlqReader.CommitMessages(ctx, msg)
			continue
		}

		originalType := events.EventType(env.Type)
		originalTopic := topicNameFor(b.config.TopicPrefix, originalType)
		if err := b.writer.WriteMessages(ctx, kafka.Message{
			Topic: originalTopic,
			Key:   []byte(env.Type),
			Value: msg.Value,
			Time:  time.Now(),
		}); err != nil {
			b.logger.Error("failed to republish DLQ message", "error", err, "event_type", originalType, "topic", originalTopic)
			return
		}

		_ = dlqReader.CommitMessages(ctx, msg)
	}
}

func parseBrokers(brokers string) []string {
	parts := strings.Split(brokers, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func topicNameFor(prefix string, eventType events.EventType) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "fintech.events"
	}
	return fmt.Sprintf("%s.%s", prefix, strings.ToLower(eventType.String()))
}

func dlqTopicNameFor(prefix string, eventType events.EventType) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "fintech.events"
	}
	return fmt.Sprintf("%s.dlq.%s", prefix, strings.ToLower(eventType.String()))
}

func errorsIsContextCanceled(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "context canceled") || strings.Contains(err.Error(), "operation was canceled"))
}

func executeHandlers(
	ctx context.Context,
	logger *slog.Logger,
	eventType events.EventType,
	evt events.Event,
	handlers []eventbus.HandlerFunc,
	msgID string,
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
				logger.Error("handler error", "error", err, "event_type", eventType, "msg_id", msgID)
			}
		}(handler)
	}

	wg.Wait()
	return success
}

var _ eventbus.Bus = (*KafkaEventBus)(nil)
