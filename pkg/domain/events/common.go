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

// Event represents a domain event in the common package.
// The generic type T is used to specify the concrete event type.
type Event interface {
	// Type returns string of the event type.
	// This is used for type-safe event registration and dispatching.
	Type() string
}
