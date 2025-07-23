package eventbus

import (
	"context"

	"github.com/amirasaad/fintech/pkg/domain"
)

// Bus defines a registry-based event bus for flexible event-driven flows.
type Bus interface {
	Register(eventType string, handler HandlerFunc)
	Emit(ctx context.Context, event domain.Event) error
}

// HandlerFunc is a generic event handler function for registry-based event buses.
type HandlerFunc func(ctx context.Context, event domain.Event) error
