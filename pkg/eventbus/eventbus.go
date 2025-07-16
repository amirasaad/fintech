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
