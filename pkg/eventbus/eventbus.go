package eventbus

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
)

// DomainEvent is a marker interface for all domain events.

// EventBus defines the contract for publishing and subscribing to domain events.
type EventBus interface {
	Publish(ctx context.Context, event domain.Event) error
	Subscribe(eventType string, handler func(context.Context, domain.Event))
}

// RegistryEventBus defines a registry-based event bus for flexible event-driven flows.
type RegistryEventBus interface {
	Register(eventType string, handler HandlerFunc)
	Emit(ctx context.Context, event Event) error
}

// Event is a generic event interface for registry-based event buses.
type Event interface {
	Type() string
}

// HandlerFunc is a generic event handler function for registry-based event buses.
type HandlerFunc func(ctx context.Context, event Event) error
