package eventbus

import (
	"context"
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

// Publish dispatches the event to all registered handlers for its type.
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
}

// NewMemoryRegistryEventBus creates a new registry-based in-memory event bus.
func NewMemoryRegistryEventBus() *MemoryRegistryEventBus {
	return &MemoryRegistryEventBus{
		handlers: make(map[string][]eventbus.HandlerFunc),
	}
}

func (b *MemoryRegistryEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *MemoryRegistryEventBus) Emit(ctx context.Context, event domain.Event) error {
	handlers := b.handlers[event.Type()]
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// Ensure MemoryRegistryEventBus implements the Bus interface.
var _ eventbus.Bus = (*MemoryRegistryEventBus)(nil)
