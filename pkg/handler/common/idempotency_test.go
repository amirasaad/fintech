package common

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyTracker(t *testing.T) {
	t.Parallel()

	t.Run("Store and check processed key", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		key := "test-key-1"

		// Initially not processed
		assert.False(t, tracker.IsProcessed(key))

		// Store it
		tracker.Store(key)

		// Should be processed now
		assert.True(t, tracker.IsProcessed(key))
	})

	t.Run("Delete removes key", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		key := "test-key-2"

		tracker.Store(key)
		assert.True(t, tracker.IsProcessed(key))

		tracker.Delete(key)

		assert.False(t, tracker.IsProcessed(key))
	})
}

func TestWithIdempotency(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()

	t.Run("executes handler when key is empty", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		executed := false
		handler := func(ctx context.Context, e events.Event) error {
			executed = true
			return nil
		}
		keyExtractor := func(e events.Event) string {
			return "" // Empty key
		}

		wrapped := WithIdempotency(handler, tracker, keyExtractor, "test-handler", logger)
		event := &testEvent{id: uuid.New()}

		err := wrapped(ctx, event)

		require.NoError(t, err)
		assert.True(t, executed, "handler should have been executed")
	})

	t.Run("skips handler when event already processed", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		executed := false
		handler := func(ctx context.Context, e events.Event) error {
			executed = true
			return nil
		}
		key := "test-key-3"
		keyExtractor := func(e events.Event) string {
			return key
		}

		wrapped := WithIdempotency(handler, tracker, keyExtractor, "test-handler", logger)
		event := &testEvent{id: uuid.New()}

		// First call - should execute
		err1 := wrapped(ctx, event)
		require.NoError(t, err1)
		assert.True(t, executed, "handler should have been executed on first call")

		// Reset flag
		executed = false

		// Second call - should be skipped
		err2 := wrapped(ctx, event)
		require.NoError(t, err2)
		assert.False(t, executed, "handler should not have been executed on second call")
	})

	t.Run("removes key from tracker when handler fails", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		handlerErr := errors.New("handler error")
		handler := func(ctx context.Context, e events.Event) error {
			return handlerErr
		}
		key := "test-key-4"
		keyExtractor := func(e events.Event) string {
			return key
		}

		wrapped := WithIdempotency(handler, tracker, keyExtractor, "test-handler", logger)
		event := &testEvent{id: uuid.New()}

		// First call - should fail
		err := wrapped(ctx, event)
		require.Error(t, err)
		assert.Equal(t, handlerErr, err)

		// Key should be removed, allowing retry
		assert.False(t, tracker.IsProcessed(key), "key should be removed after handler failure")
	})

	t.Run("keeps key in tracker when handler succeeds", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		handler := func(ctx context.Context, e events.Event) error {
			return nil
		}
		key := "test-key-5"
		keyExtractor := func(e events.Event) string {
			return key
		}

		wrapped := WithIdempotency(handler, tracker, keyExtractor, "test-handler", logger)
		event := &testEvent{id: uuid.New()}

		err := wrapped(ctx, event)
		require.NoError(t, err)

		// Key should remain in tracker
		assert.True(
			t,
			tracker.IsProcessed(key),
			"key should remain in tracker after successful handler")
	})

	t.Run("concurrent duplicate processing does not silently drop failures", func(t *testing.T) {
		t.Parallel()

		tracker := NewIdempotencyTracker()
		const n = 20
		key := "test-key-concurrent"
		var extracted int32
		allExtracted := make(chan struct{})
		var returned int32
		allReturned := make(chan struct{})
		keyExtractor := func(e events.Event) string {
			if atomic.AddInt32(&extracted, 1) == n {
				close(allExtracted)
			}
			<-allExtracted
			if atomic.AddInt32(&returned, 1) == n {
				close(allReturned)
			}
			return key
		}

		handlerStarted := make(chan struct{})
		release := make(chan struct{})

		var calls int32
		handlerErr := errors.New("handler failed")
		handler := func(ctx context.Context, e events.Event) error {
			if atomic.AddInt32(&calls, 1) == 1 {
				close(handlerStarted)
			}
			<-release
			return handlerErr
		}

		wrapped := WithIdempotency(handler, tracker, keyExtractor, "test-handler", logger)
		event := &testEvent{id: uuid.New()}

		begin := make(chan struct{})
		var ready sync.WaitGroup
		ready.Add(n)
		var wg sync.WaitGroup
		wg.Add(n)
		errs := make([]error, n)
		for i := 0; i < n; i++ {
			i := i
			go func() {
				defer wg.Done()
				ready.Done()
				<-begin
				errs[i] = wrapped(ctx, event)
			}()
		}
		ready.Wait()
		close(begin)

		<-handlerStarted
		<-allExtracted
		<-allReturned
		time.Sleep(10 * time.Millisecond)
		close(release)
		wg.Wait()

		assert.EqualValues(
			t,
			1,
			atomic.LoadInt32(&calls), "handler should run at most once for concurrent duplicates")
		for i := range n {
			assert.ErrorIs(
				t,
				errs[i],
				handlerErr, "concurrent duplicate must observe the same failure (no silent skip)")
		}
	})

	t.Run("uses default logger when nil logger provided", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		executed := false
		handler := func(ctx context.Context, e events.Event) error {
			executed = true
			return nil
		}
		keyExtractor := func(e events.Event) string {
			return "test-key-6"
		}

		// Pass nil logger - should use slog.Default()
		wrapped := WithIdempotency(handler, tracker, keyExtractor, "test-handler", nil)
		event := &testEvent{id: uuid.New()}

		err := wrapped(ctx, event)

		require.NoError(t, err)
		assert.True(t, executed)
	})
}

// Test event implementation for testing
type testEvent struct {
	id uuid.UUID
}

func (e *testEvent) Type() string {
	return "test.event"
}
