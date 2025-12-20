//go:build kafka
// +build kafka

package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	testcontainerskafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

type KafkaTestEvent struct {
	Message string
}

// Type returns the event type for KafkaTestEvent.
func (e *KafkaTestEvent) Type() string {
	return "test.event"
}

// setupKafkaBus starts a Kafka container using testcontainers-go and returns a
// KafkaEventBus and a cleanup function.
func setupKafkaBus(tb testing.TB) (*KafkaEventBus, func()) {
	tb.Helper()

	if !dockerIsReachable() {
		tb.Skip("docker is not reachable")
	}

	defer func() {
		if r := recover(); r != nil {
			tb.Skipf("skipping kafka integration test: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	kafkaContainer, err := testcontainerskafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	if err != nil {
		tb.Fatalf("failed to start kafka container: %v", err)
	}

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		tb.Fatalf("failed to get kafka brokers: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	bus, err := NewWithKafka(strings.Join(brokers, ","), logger, nil)
	if err != nil {
		tb.Fatalf("failed to create kafka event bus: %v", err)
	}

	cleanup := func() {
		_ = bus.Close()
		_ = kafkaContainer.Terminate(context.Background())
	}

	return bus, cleanup
}

func dockerIsReachable() bool {
	host := os.Getenv("DOCKER_HOST")
	if strings.HasPrefix(host, "unix://") {
		return canDialUnix(strings.TrimPrefix(host, "unix://"))
	}
	if host != "" {
		return true
	}
	if canDialUnix("/var/run/docker.sock") {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	return canDialUnix(home + "/.docker/run/docker.sock")
}

func canDialUnix(path string) bool {
	if path == "" {
		return false
	}
	conn, err := net.DialTimeout("unix", path, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestKafkaBusHandlerReceivesEvent(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event { return &KafkaTestEvent{} }
	bus, cleanup := setupKafkaBus(t)
	defer cleanup()

	received := make(chan string, 1)
	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		te := e.(*KafkaTestEvent)
		received <- te.Message
		return nil
	})

	require.NoError(t, bus.Emit(context.Background(), &KafkaTestEvent{Message: "hello"}))

	select {
	case msg := <-received:
		require.Equal(t, "hello", msg)
	case <-time.After(10 * time.Second):
		t.Fatal("handler did not receive event in time")
	}
}

func TestKafkaBusDLQ(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event { return &KafkaTestEvent{} }
	bus, cleanup := setupKafkaBus(t)
	defer cleanup()

	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		return fmt.Errorf("simulated failure")
	})

	require.NoError(t, bus.Emit(context.Background(), &KafkaTestEvent{Message: "should go to DLQ"}))

	dlqReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     bus.brokers,
		Topic:       dlqTopicNameFor(bus.config.TopicPrefix, events.EventType("test.event")),
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
	})
	defer func() { _ = dlqReader.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for {
		msg, err := dlqReader.FetchMessage(ctx)
		if err != nil {
			t.Fatalf("failed to fetch dlq message: %v", err)
		}
		if len(msg.Value) == 0 {
			continue
		}
		return
	}
}

func TestKafkaBusDLQRetry(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event { return &KafkaTestEvent{} }
	bus, cleanup := setupKafkaBus(t)
	defer cleanup()

	fail := true
	received := make(chan string, 1)
	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		if fail {
			return fmt.Errorf("temporary failure")
		}
		te := e.(*KafkaTestEvent)
		received <- te.Message
		return nil
	})

	require.NoError(t, bus.Emit(context.Background(), &KafkaTestEvent{Message: "retry me"}))

	time.Sleep(2 * time.Second)
	fail = false

	bus.processAllDLQs(context.Background())

	select {
	case msg := <-received:
		require.Equal(t, "retry me", msg)
	case <-time.After(15 * time.Second):
		t.Fatal("DLQ retry did not republish message in time")
	}
}
