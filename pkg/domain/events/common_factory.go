package events

import (
	"github.com/google/uuid"
	"time"
)

type FlowEventOpt func(*FlowEvent)

func NewFlowEvent(opts ...FlowEventOpt) *FlowEvent {
	e := &FlowEvent{
		ID:            uuid.New(),
		FlowType:      "",
		UserID:        uuid.New(),
		AccountID:     uuid.New(),
		CorrelationID: uuid.New(),
		Timestamp:     time.Now(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *FlowEvent) WithUserID(userID uuid.UUID) *FlowEvent {
	e.UserID = userID
	return e
}

func (e *FlowEvent) WithID(id uuid.UUID) *FlowEvent {
	e.ID = id
	return e
}

func (e *FlowEvent) WithAccountID(accountID uuid.UUID) *FlowEvent {
	e.AccountID = accountID
	return e
}

func (e *FlowEvent) WithCorrelationID(correlationID uuid.UUID) *FlowEvent {
	e.CorrelationID = correlationID
	return e
}

func (e *FlowEvent) WithFlowType(flowType string) *FlowEvent {
	e.FlowType = flowType
	return e
}
