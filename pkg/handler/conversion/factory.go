package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain/events"
)

// EventFactory defines the interface for creating the next event after a currency conversion.
// Each business flow (deposit, withdraw, etc.) will have its own implementation of this factory
// to construct the appropriate flow-specific event after currency conversion.
type EventFactory interface {
	// CreateNextEvent creates and returns the event that should be emitted after a successful currency conversion.
	// It takes a CurrencyConverted event that contains all the necessary information
	// including the converted amount to create the next event in the flow.
	CreateNextEvent(convertedEvent *events.CurrencyConverted) events.Event
}
