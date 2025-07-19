package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/amirasaad/fintech/pkg/domain"
)

type SimpleEventBus struct {
	handlers map[string][]func(context.Context, domain.Event)
	mu       sync.RWMutex
}

func NewSimpleEventBus() *SimpleEventBus {
	return &SimpleEventBus{handlers: make(map[string][]func(context.Context, domain.Event))}
}

func (b *SimpleEventBus) Publish(ctx context.Context, event domain.Event) error {
	// Debug log: print concrete type and event type string
	slog.Debug("EventBus.Publish", "event_type", event.Type(), "concrete_type", fmt.Sprintf("%T", event))
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, handler := range b.handlers[event.Type()] {
		handler(ctx, event)
	}
	return nil
}

func (b *SimpleEventBus) Subscribe(eventType string, handler func(context.Context, domain.Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}
