package events

import "github.com/amirasaad/fintech/pkg/queries"

type AccountQuerySucceededEvent struct {
	Result queries.GetAccountResult
}

type AccountQueryFailedEvent struct {
	Query  queries.GetAccountQuery
	Reason string
}

type AccountValidatedEvent struct {
	AccountID string
	UserID    string
	Amount    int64
	Currency  string
}

type AccountValidationFailedEvent struct {
	AccountID string
	UserID    string
	Reason    string
}

func (e AccountQuerySucceededEvent) EventType() string   { return "AccountQuerySucceededEvent" }
func (e AccountQueryFailedEvent) EventType() string      { return "AccountQueryFailedEvent" }
func (e AccountValidatedEvent) EventType() string        { return "AccountValidatedEvent" }
func (e AccountValidationFailedEvent) EventType() string { return "AccountValidationFailedEvent" }
