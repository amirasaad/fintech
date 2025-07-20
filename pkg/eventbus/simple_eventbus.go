package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain"
)

type SimpleEventBus struct {
	handlers map[string][]HandlerFunc
	mu       sync.RWMutex
}

func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{handlers: make(map[string][]HandlerFunc)}
}

func (b *SimpleEventBus) Emit(ctx context.Context, event domain.Event) error {
	// Debug log: print concrete type and event type string
	slog.Debug("EventBus.Publish", "event_type", event.Type(), "concrete_type", fmt.Sprintf("%T", event))
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, handler := range b.handlers[event.Type()] {
		handler(ctx, event) //nolint:errcheck
	}
	return nil
}

func (b *SimpleEventBus) Subscribe(eventType string, handler HandlerFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *SimpleEventBus) Register(eventType string, handler HandlerFunc) {
	if b.handlers == nil {
		b.handlers = make(map[string][]HandlerFunc)
	}
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}
