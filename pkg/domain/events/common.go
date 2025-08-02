package events

import (
	"time"

	"github.com/google/uuid"
)

type FlowEvent struct {
	ID            uuid.UUID
	FlowType      string
	UserID        uuid.UUID
	AccountID     uuid.UUID
	CorrelationID uuid.UUID
	Timestamp     time.Time
}

// Validator defines an interface for objects that can be validated.
type Validator interface {
	Validate() error
}
