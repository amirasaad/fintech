package eventbus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestEvent struct {
	Message string
}

func (e *TestEvent) Type() string {
	return "test.event"
}

// setupRedisBus starts a Redis container using testcontainers-go and returns a
// RedisEventBus and a cleanup function.

func setupRedisBus(tb testing.TB) (*RedisEventBus, func()) {
	tb.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:7.0.5",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		tb.Fatalf("Failed to start container: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		tb.Fatalf("Failed to get mapped port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		tb.Fatalf("Failed to get container host: %v", err)
	}

	url := "redis://" + host + ":" + port.Port()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Use default config for tests
	bus, err := NewWithRedis(url, logger, nil)
	if err != nil {
		tb.Fatalf("Failed to create Redis event bus: %v", err)
	}

	cleanup := func() {
		_ = container.Terminate(ctx)
	}
	return bus, cleanup
}

func TestRedisBusHandlerReceivesEvent(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event { return &TestEvent{} }
	bus, cleanup := setupRedisBus(t)
	defer cleanup()

	received := make(chan string, 1)
	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		te := e.(*TestEvent)
		received <- te.Message
		return nil
	})

	ctx := context.Background()
	err := bus.Emit(ctx, &TestEvent{Message: "hello"})
	require.NoError(t, err)

	select {
	case msg := <-received:
		require.Equal(t, "hello", msg)
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not receive event in time")
	}
}

func TestRedisBusMultipleEvents(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event { return &TestEvent{} }
	bus, cleanup := setupRedisBus(t)
	defer cleanup()

	count := 0
	done := make(chan struct{})
	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		count++
		if count == 3 {
			close(done)
		}
		return nil
	})

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		err := bus.Emit(ctx, &TestEvent{Message: fmt.Sprintf("msg %d", i)})
		require.NoError(t, err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("not all events were received")
	}
}

func TestRedisBusDLQ(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event {
		return &TestEvent{}
	}
	bus, cleanup := setupRedisBus(t)
	defer cleanup()

	ctx := context.Background()

	// Register a handler that always fails
	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		return fmt.Errorf("simulated failure")
	})

	// Emit an event
	err := bus.Emit(ctx, &TestEvent{Message: "should go to DLQ"})
	require.NoError(t, err)

	// Allow some time for the handler to process and fail
	time.Sleep(2 * time.Second)

	// Connect to Redis to check the DLQ
	// We reconstruct the Redis URL as in setupRedisBus
	// The stream name is "dlq:test:event"

	dlqStream := dlqStreamName("test.event")
	res, err := bus.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{dlqStream, "0"},
		Count:   1,
		Block:   time.Second,
	}).Result()

	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Messages, 1)
}

// TestRedisBusDLQRetry verifies that DLQ retry republishes messages
// to the original stream and handlers can successfully consume them after a failure.
func TestRedisBusDLQRetry(t *testing.T) {
	events.EventTypes["test.event"] = func() events.Event { return &TestEvent{} }
	bus, cleanup := setupRedisBus(t)
	defer cleanup()

	ctx := context.Background()

	// 1) Register a handler that fails initially so the message goes to DLQ
	fail := true
	received := make(chan string, 1)
	bus.Register("test.event", func(ctx context.Context, e events.Event) error {
		if fail {
			return fmt.Errorf("temporary failure")
		}
		te := e.(*TestEvent)
		received <- te.Message
		return nil
	})

	// Emit the event
	require.NoError(t, bus.Emit(ctx, &TestEvent{Message: "retry me"}))

	// Wait for it to reach DLQ
	time.Sleep(1 * time.Second)

	// 2) Switch the behavior to succeed on retry
	fail = false

	// 3) Trigger DLQ processing explicitly (avoids waiting for the periodic worker)
	bus.processAllDLQs(ctx)

	select {
	case msg := <-received:
		require.Equal(t, "retry me", msg)
	case <-time.After(3 * time.Second):
		t.Fatal("DLQ retry did not republish message in time")
	}
}
