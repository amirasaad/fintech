package common

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

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
		_, already := tracker.processed.LoadOrStore(key, struct{}{})
		assert.False(t, already)

		// Store it
		tracker.Store(key)

		// Should be processed now
		_, already = tracker.processed.LoadOrStore(key, struct{}{})
		assert.True(t, already)
	})

	t.Run("Delete removes key", func(t *testing.T) {
		t.Parallel()
		tracker := NewIdempotencyTracker()
		key := "test-key-2"

		tracker.Store(key)
		_, already := tracker.processed.LoadOrStore(key, struct{}{})
		assert.True(t, already)

		tracker.Delete(key)

		_, already = tracker.processed.LoadOrStore(key, struct{}{})
		assert.False(t, already)
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
		_, already := tracker.processed.LoadOrStore(key, struct{}{})
		assert.False(t, already, "key should be removed after handler failure")
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
		_, already := tracker.processed.LoadOrStore(key, struct{}{})
		assert.True(t, already, "key should remain in tracker after successful handler")
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
