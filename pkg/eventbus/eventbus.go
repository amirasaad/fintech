package eventbus

import (
	"github.com/amirasaad/fintech/pkg/domain"
)

// DomainEvent is a marker interface for all domain events.

// EventBus defines the contract for publishing and subscribing to domain events.
type EventBus interface {
	Publish(event domain.Event) error
	Subscribe(eventType string, handler func(domain.Event))
}
