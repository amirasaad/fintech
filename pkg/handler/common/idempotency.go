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

func (t *IdempotencyTracker) IsProcessed(key string) bool {
	_, ok := t.processed.Load(key)
	return ok
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
		if tracker.IsProcessed(key) {
			log.Info("🔁 [SKIP] Event already processed")
			return nil
		}

		_, err, _ := tracker.inflight.Do(key, func() (any, error) {
			if tracker.IsProcessed(key) {
				return nil, nil
			}

			if err := handler(ctx, e); err != nil {
				return nil, err
			}

			tracker.Store(key)
			return nil, nil
		})
		if err != nil {
			return err
		}

		return nil
	}
}
