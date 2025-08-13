package eventbus

import (
	"context"
	"testing"
)

type benchmarkEvent struct {
	ID int
}

func (e *benchmarkEvent) Type() string {
	return "benchmark.event"
}

func BenchmarkRedisEmit(b *testing.B) {
	bus, cleanup := setupRedisBus(b)
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
	bus, cleanup := setupRedisBus(b)
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
