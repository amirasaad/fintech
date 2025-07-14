package eventbus

import (
	"fmt"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/eventbus"
)

// MemoryEventBus is a simple in-memory implementation of the EventBus interface.
type MemoryEventBus struct {
	Events []account.PaymentEvent
}

// PublishPaymentEvent logs and records the event in memory.
func (b *MemoryEventBus) PublishPaymentEvent(event account.PaymentEvent) error {
	fmt.Printf("Event published: %+v\n", event)
	b.Events = append(b.Events, event)
	return nil
}

// Ensure MemoryEventBus implements the EventBus interface.
var _ eventbus.EventBus = (*MemoryEventBus)(nil)
