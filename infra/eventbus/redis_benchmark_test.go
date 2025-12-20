//go:build redis
// +build redis

package eventbus

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type benchmarkEvent struct {
	ID int
}

func (e *benchmarkEvent) Type() string {
	return "benchmark.event"
}

func setupRedisBusForBenchmark(b *testing.B) (*RedisEventBus, func()) {
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
		b.Fatalf("Failed to start container: %v", err)
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		b.Fatalf("Failed to get mapped port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		b.Fatalf("Failed to get container host: %v", err)
	}

	url := "redis://" + host + ":" + port.Port()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Use default config for benchmarks
	bus, err := NewWithRedis(url, logger, nil)
	if err != nil {
		b.Fatalf("Failed to create Redis event bus: %v", err)
	}

	cleanup := func() {
		_ = container.Terminate(ctx)
	}
	return bus, cleanup
}

func BenchmarkRedisEmit(b *testing.B) {
	bus, cleanup := setupRedisBusForBenchmark(b)
	defer cleanup()

	ctx := context.Background()
	event := &benchmarkEvent{}

	for i := 0; b.Loop(); i++ {
		event.ID = i
		_ = bus.Emit(ctx, event)
	}
}

func BenchmarkRedisEmitParallel(b *testing.B) {
	ctx := context.Background()
	bus, cleanup := setupRedisBusForBenchmark(b)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		event := &benchmarkEvent{}
		i := 0
		for pb.Next() {
			event.ID = i
			_ = bus.Emit(ctx, event)
			i++
		}
	})
}
