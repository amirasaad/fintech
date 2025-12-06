package common

import (
	"context"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// KeyExtractor extracts an idempotency key from an event
type KeyExtractor func(events.Event) string

// IdempotencyTracker tracks processed events by key
type IdempotencyTracker struct {
	processed sync.Map
}

// NewIdempotencyTracker creates a new idempotency tracker
func NewIdempotencyTracker() *IdempotencyTracker {
	return &IdempotencyTracker{}
}

// Store marks a key as processed
func (t *IdempotencyTracker) Store(key string) {
	t.processed.Store(key, struct{}{})
}

// Delete removes a key from the tracker
func (t *IdempotencyTracker) Delete(key string) {
	t.processed.Delete(key)
}

// WithIdempotency wraps a handler with idempotency checking middleware.
// The middleware checks if the event has been processed before calling the handler,
// and marks it as processed after successful execution.
func WithIdempotency(
	handler eventbus.HandlerFunc,
	tracker *IdempotencyTracker,
	keyExtractor KeyExtractor,
	handlerName string,
	logger *slog.Logger,
) eventbus.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, e events.Event) error {
		key := keyExtractor(e)
		if key == "" {
			// No key extracted, proceed without idempotency check
			return handler(ctx, e)
		}

		log := logger.With(
			"handler", handlerName,
			"event_type", e.Type(),
			"idempotency_key", key,
		)

		// Check if already processed (before calling handler)
		if _, already := tracker.processed.LoadOrStore(key, struct{}{}); already {
			log.Info("🔁 [SKIP] Event already processed")
			return nil
		}

		// Execute handler
		err := handler(ctx, e)

		// If handler failed, remove from tracker to allow retry
		if err != nil {
			tracker.processed.Delete(key)
			return err
		}

		// Handler succeeded, keep marked as processed
		return nil
	}
}
