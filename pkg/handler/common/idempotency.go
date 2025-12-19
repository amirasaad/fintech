package common

import (
	"context"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"golang.org/x/sync/singleflight"
)

// KeyExtractor extracts an idempotency key from an event
type KeyExtractor func(events.Event) string

// IdempotencyTracker tracks processed events by key
type IdempotencyTracker struct {
	processed sync.Map
	inflight  singleflight.Group
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
		if _, already := tracker.processed.Load(key); already {
			log.Info("🔁 [SKIP] Event already processed")
			return nil
		}

		// Ensure only one goroutine processes a given key at a time.
		// Other goroutines will wait and observe the same success/failure result,
		// preventing silent drops when the in-flight attempt fails.
		_, err, _ := tracker.inflight.Do(key, func() (any, error) {
			// Another goroutine may have completed successfully while we waited.
			if _, already := tracker.processed.Load(key); already {
				return nil, nil
			}

			// Execute handler
			if err := handler(ctx, e); err != nil {
				return nil, err
			}

			// Handler succeeded, mark as processed
			tracker.processed.Store(key, struct{}{})
			return nil, nil
		})
		if err != nil {
			// Ensure key is not left marked as processed on failure.
			// (This is defensive; we only store on success.)
			tracker.processed.Delete(key)
			return err
		}

		// Handler succeeded (either by us or a concurrent goroutine)
		return nil
	}
}
