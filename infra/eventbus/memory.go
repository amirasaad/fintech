package eventbus

import (
	"context"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

const EventDepthKey = "eventDepth"
const MaxEventDepth = 10

// MemoryEventBus is a simple in-memory implementation of the EventBus interface.
type MemoryEventBus struct {
	handlers  map[string][]eventbus.HandlerFunc
	mu        sync.RWMutex
	logger    *slog.Logger
	published []common.Event // Added for testing purposes
}

// NewWithMemory creates a new in-memory event bus for event-driven communication.
func NewWithMemory(logger *slog.Logger) *MemoryEventBus {
	return &MemoryEventBus{
		handlers:  make(map[string][]eventbus.HandlerFunc),
		logger:    logger.With("bus", "memory"),
		published: make([]common.Event, 0), // Initialize the slice
	}
}

// Register i registers a handler for a specific event type.
func (b *MemoryEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Emit dispatches the event to all registered handlers for its type.
func (b *MemoryEventBus) Emit(ctx context.Context, event common.Event) error {
	eventType := event.Type()
	b.mu.RLock()
	handlers := b.handlers[eventType]
	b.mu.RUnlock()

	// Store the published event for testing
	b.mu.Lock()
	b.published = append(b.published, event)
	b.mu.Unlock()

	for _, handler := range handlers {
		handler(ctx, event) //nolint:errcheck
	}
	return nil
}

// ClearPublished clears the list of published events. This is useful for testing.
func (b *MemoryEventBus) ClearPublished() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = make([]common.Event, 0)
}

// GetPublished returns the list of published events. This is useful for testing.
func (b *MemoryEventBus) Published() []common.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.published
}

// Ensure MemoryEventBus implements the EventBus interface.
var _ eventbus.Bus = (*MemoryEventBus)(nil)

// MemoryAsyncEventBus is a registry-based in-memory event bus implementation.
type MemoryAsyncEventBus struct {
	handlers map[string][]eventbus.HandlerFunc
	mu       sync.RWMutex
	eventCh  chan struct {
		ctx   context.Context
		event common.Event
	}
	log *slog.Logger
}

// NewWithMemoryAsync creates a new registry-based in-memory event bus.
func NewWithMemoryAsync(logger *slog.Logger) *MemoryAsyncEventBus {
	b := &MemoryAsyncEventBus{
		handlers: make(map[string][]eventbus.HandlerFunc),
		eventCh: make(chan struct {
			ctx   context.Context
			event common.Event
		}, 100),
	}
	go b.process()
	b.log = logger.With("event-bus", "memory")
	return b
}

func (b *MemoryAsyncEventBus) Register(eventType string, handler eventbus.HandlerFunc) {

	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.mu.Unlock()
}

func (b *MemoryAsyncEventBus) Emit(ctx context.Context, event common.Event) error {
	b.eventCh <- struct {
		ctx   context.Context
		event common.Event
	}{ctx, event}
	return nil
}

func (b *MemoryAsyncEventBus) process() {
	for w := range b.eventCh {
		go func(w struct {
			ctx   context.Context
			event common.Event
		}) {
			b.mu.RLock()
			handlers := append([]eventbus.HandlerFunc{}, b.handlers[w.event.Type()]...)
			b.mu.RUnlock()
			for _, handler := range handlers {
				func() {
					defer func() {
						if r := recover(); r != nil {
							b.log.Error("panic recovered in event handler", "type", w.event.Type(), "event", w.event, "panic", r)
						}
					}()
					if err := handler(w.ctx, w.event); err != nil {
						b.log.Error("failed to process event", "type", w.event.Type(), "event", w.event, "error", err)
					}
				}()
			}
		}(w)
	}
}

// Ensure MemoryRegistryEventBus implements the Bus interface.
var _ eventbus.Bus = (*MemoryAsyncEventBus)(nil)
