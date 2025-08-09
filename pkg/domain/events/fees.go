package events

import (
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
)

// FeesCalculated is emitted after all fees for a transaction have been calculated.
type FeesCalculated struct {
	FlowEvent
	TransactionID uuid.UUID
	Fee           account.Fee
}

func (e FeesCalculated) Type() string { return EventTypeFeesCalculated.String() }
