package eventbus

import (
	"context"
	"log/slog"
	"reflect"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

const EventDepthKey = "eventDepth"
const MaxEventDepth = 10

// MemoryEventBus is a simple in-memory implementation of the EventBus interface.
type MemoryEventBus struct {
	handlers map[string][]eventbus.HandlerFunc
	mu       sync.RWMutex
}

// NewMemoryEventBus creates a new in-memory event bus for event-driven communication.
func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		handlers: make(map[string][]eventbus.HandlerFunc),
	}
}

// Register i registers a handler for a specific event type.
func (b *MemoryEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Emit dispatches the event to all registered handlers for its type.
func (b *MemoryEventBus) Emit(ctx context.Context, event domain.Event) error {
	eventType := reflect.TypeOf(event).Name()
	b.mu.RLock()
	handlers := b.handlers[eventType]
	b.mu.RUnlock()
	for _, handler := range handlers {
		handler(ctx, event) //nolint:errcheck
	}
	return nil
}

// Ensure MemoryEventBus implements the EventBus interface.
var _ eventbus.Bus = (*MemoryEventBus)(nil)

// MemoryRegistryEventBus is a registry-based in-memory event bus implementation.
type MemoryRegistryEventBus struct {
	handlers map[string][]eventbus.HandlerFunc
	mu       sync.RWMutex
	eventCh  chan struct {
		ctx   context.Context
		event domain.Event
	}
	log *slog.Logger
}

// NewMemoryRegistryEventBus creates a new registry-based in-memory event bus.
func NewMemoryRegistryEventBus(logger *slog.Logger) *MemoryRegistryEventBus {
	b := &MemoryRegistryEventBus{
		handlers: make(map[string][]eventbus.HandlerFunc),
		eventCh: make(chan struct {
			ctx   context.Context
			event domain.Event
		}, 100),
	}
	go b.process()
	b.log = logger.With("event-bus", "memory")
	return b
}

func (b *MemoryRegistryEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.mu.Unlock()
}

func (b *MemoryRegistryEventBus) Emit(ctx context.Context, event domain.Event) error {
	b.eventCh <- struct {
		ctx   context.Context
		event domain.Event
	}{ctx, event}
	return nil
}

func (b *MemoryRegistryEventBus) process() {
	for w := range b.eventCh {
		go func(w struct {
			ctx   context.Context
			event domain.Event
		}) {
			b.mu.RLock()
			handlers := append([]eventbus.HandlerFunc{}, b.handlers[w.event.Type()]...)
			b.mu.RUnlock()
			for _, handler := range handlers {
				if err := handler(w.ctx, w.event); err != nil {
					b.log.Error("failed to process event", "type", w.event.Type(), "event", w.event, "error", err)
					break
				}
			}
		}(w)
	}
}

// Ensure MemoryRegistryEventBus implements the Bus interface.
var _ eventbus.Bus = (*MemoryRegistryEventBus)(nil)
