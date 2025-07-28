package events

import (
	"github.com/google/uuid"
)

type FlowEvent struct {
	FlowType      string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	CorrelationID uuid.UUID
}

// Validatable defines an interface for objects that can be validated.
type Validatable interface {
	Validate() error
}
