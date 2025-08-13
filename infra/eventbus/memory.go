package eventbus

import (
	"context"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/eventbus"
)

const EventDepthKey = "eventDepth"
const MaxEventDepth = 10

// MemoryEventBus is a simple in-memory implementation of the EventBus interface.
type MemoryEventBus struct {
	handlers  map[events.EventType][]eventbus.HandlerFunc
	mu        sync.RWMutex
	logger    *slog.Logger
	published []events.Event // Added for testing purposes
}

// NewWithMemory creates a new in-memory event bus for event-driven
// communication.
func NewWithMemory(logger *slog.Logger) *MemoryEventBus {
	return &MemoryEventBus{
		handlers:  make(map[events.EventType][]eventbus.HandlerFunc),
		logger:    logger.With("bus", "memory"),
		published: make([]events.Event, 0), // Initialize the slice
	}
}

// Register registers a handler for a specific event type.
func (b *MemoryEventBus) Register(
	eventType events.EventType,
	handler eventbus.HandlerFunc,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Emit dispatches the event to all registered handlers for its type.
func (b *MemoryEventBus) Emit(ctx context.Context, event events.Event) error {
	eventType := events.EventType(event.Type())
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

// ClearPublished clears the list of published events.
// This is useful for testing.
func (b *MemoryEventBus) ClearPublished() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = make([]events.Event, 0)
}

// Published returns the list of published events. This is useful for testing.
func (b *MemoryEventBus) Published() []events.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.published
}

// Ensure MemoryEventBus implements the EventBus interface.
var _ eventbus.Bus = (*MemoryEventBus)(nil)

// MemoryAsyncEventBus is a registry-based in-memory event bus implementation.
type MemoryAsyncEventBus struct {
	handlers map[events.EventType][]eventbus.HandlerFunc
	mu       sync.RWMutex
	eventCh  chan struct {
		ctx   context.Context
		event events.Event
	}
	log *slog.Logger
}

// NewWithMemoryAsync creates a new registry-based in-memory event bus.
func NewWithMemoryAsync(logger *slog.Logger) *MemoryAsyncEventBus {
	b := &MemoryAsyncEventBus{
		handlers: make(map[events.EventType][]eventbus.HandlerFunc),
		eventCh: make(chan struct {
			ctx   context.Context
			event events.Event
		}, 100),
	}
	go b.process()
	b.log = logger.With("event-bus", "memory")
	return b
}

func (b *MemoryAsyncEventBus) Register(
	eventType events.EventType,
	handler eventbus.HandlerFunc,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *MemoryAsyncEventBus) Emit(
	ctx context.Context,
	event events.Event,
) error {
	b.eventCh <- struct {
		ctx   context.Context
		event events.Event
	}{ctx, event}
	return nil
}

// getHandlers returns a copy of the handlers for the given event type.
func (b *MemoryAsyncEventBus) getHandlers(
	eventType events.EventType,
) []eventbus.HandlerFunc {
	b.mu.RLock()
	defer b.mu.RUnlock()
	handlers := make([]eventbus.HandlerFunc, len(b.handlers[eventType]))
	copy(handlers, b.handlers[eventType])
	return handlers
}

func (b *MemoryAsyncEventBus) process() {
	for item := range b.eventCh {
		eventType := events.EventType(item.event.Type())
		handlers := b.getHandlers(eventType)
		for _, handler := range handlers {
			go func(
				h eventbus.HandlerFunc,
				ctx context.Context,
				evt events.Event,
			) {
				if err := h(ctx, evt); err != nil {
					b.log.Error("error handling event", "error", err, "event_type", eventType)
				}
			}(handler, item.ctx, item.event)
		}
	}
}

// Ensure MemoryRegistryEventBus implements the Bus interface.
var _ eventbus.Bus = (*MemoryAsyncEventBus)(nil)
