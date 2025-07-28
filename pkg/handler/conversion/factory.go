package conversion

import (
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
)

// EventFactory defines the interface for creating the next event after a currency conversion.
// Each business flow (deposit, withdraw, etc.) will have its own implementation of this factory
// to construct the appropriate flow-specific "ConversionDone" event.
type EventFactory interface {
	// CreateNextEvent creates and returns the event that should be emitted after a successful currency conversion.
	CreateNextEvent(
		cre *events.ConversionRequestedEvent,
		convInfo *common.ConversionInfo,
		convertedMoney money.Money,
	) (common.Event, error)
}
