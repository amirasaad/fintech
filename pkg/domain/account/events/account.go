package events

import (
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
)

type AccountQuerySucceededEvent struct {
	Result queries.GetAccountResult
}

type AccountQueryFailedEvent struct {
	Query  queries.GetAccountQuery
	Reason string
}

type AccountValidatedEvent struct {
	AccountID uuid.UUID
	UserID    uuid.UUID
	Amount    int64
	Currency  string
}

type AccountValidationFailedEvent struct {
	AccountID uuid.UUID
	UserID    uuid.UUID
	Reason    string
}

func (e AccountQuerySucceededEvent) EventType() string   { return "AccountQuerySucceededEvent" }
func (e AccountQueryFailedEvent) EventType() string      { return "AccountQueryFailedEvent" }
func (e AccountValidatedEvent) EventType() string        { return "AccountValidatedEvent" }
func (e AccountValidationFailedEvent) EventType() string { return "AccountValidationFailedEvent" }
