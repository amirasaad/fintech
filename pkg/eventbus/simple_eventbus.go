package eventbus

import (
	"context"
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
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, handler := range b.handlers[event.EventType()] {
		handler(ctx, event)
	}
	return nil
}

func (b *SimpleEventBus) Subscribe(eventType string, handler func(context.Context, domain.Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}
