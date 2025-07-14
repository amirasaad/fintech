package eventbus

import "github.com/amirasaad/fintech/pkg/domain/account"

// EventBus defines the contract for publishing payment events.
type EventBus interface {
	PublishPaymentEvent(event account.PaymentEvent) error
}
