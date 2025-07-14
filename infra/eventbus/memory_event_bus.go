package eventbus

import (
	"reflect"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// MemoryEventBus is a simple in-memory implementation of the EventBus interface.
type MemoryEventBus struct {
	handlers map[string][]func(domain.Event)
	mu       sync.RWMutex
}

func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		handlers: make(map[string][]func(domain.Event)),
	}
}

// Subscribe registers a handler for a specific event type.
func (b *MemoryEventBus) Subscribe(eventType string, handler func(domain.Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish dispatches the event to all registered handlers for its type.
func (b *MemoryEventBus) Publish(event domain.Event) error {
	eventType := reflect.TypeOf(event).Name()
	b.mu.RLock()
	handlers := b.handlers[eventType]
	b.mu.RUnlock()
	for _, handler := range handlers {
		handler(event)
	}
	return nil
}

// Ensure MemoryEventBus implements the EventBus interface.
var _ eventbus.EventBus = (*MemoryEventBus)(nil)
