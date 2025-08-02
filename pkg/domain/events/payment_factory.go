package events

import "github.com/google/uuid"

// PaymentInitiatedOpt is a function that configures a PaymentInitiated
type PaymentInitiatedOpt func(*PaymentInitiated)

// WithPaymentTransactionID sets the transaction ID for the PaymentInitiated
func WithPaymentTransactionID(id uuid.UUID) PaymentInitiatedOpt {
	return func(e *PaymentInitiated) { e.TransactionID = id }
}

// WithPaymentID sets the payment ID for the PaymentInitiated
func WithInitiatedPaymentID(paymentID string) PaymentInitiatedOpt {
	return func(e *PaymentInitiated) { e.PaymentID = paymentID }
}

// WithPaymentStatus sets the status for the PaymentInitiated
func WithInitiatedPaymentStatus(status string) PaymentInitiatedOpt {
	return func(e *PaymentInitiated) { e.Status = status }
}

// WithFlowEvent sets the FlowEvent from an existing FlowEvent
func WithFlowEvent(flowEvent FlowEvent) PaymentInitiatedOpt {
	return func(e *PaymentInitiated) {
		e.FlowEvent = flowEvent
	}
}

// NewPaymentInitiated creates a new PaymentInitiated with the given options
func NewPaymentInitiated(flowEvent FlowEvent, opts ...PaymentInitiatedOpt) *PaymentInitiated {
	p := &PaymentInitiated{
		FlowEvent:     flowEvent,
		TransactionID: uuid.Nil,
		PaymentID:     "",
		Status:        "initiated",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type PaymentProcessedOpt func(*PaymentProcessed)

// NewPaymentProcessed creates a new PaymentProcessed with the given parameters
func NewPaymentProcessed(pi PaymentInitiated, opts ...PaymentProcessedOpt) *PaymentProcessed {
	// Create base PaymentInitiated with required fields
	pp := &PaymentProcessed{
		PaymentInitiated: pi,
	}

	// Apply any additional options
	for _, opt := range opts {
		opt(pp)
	}
	return pp

}

// PaymentCompletedOpt is a function that configures a PaymentCompletedEvent
type PaymentCompletedOpt func(*PaymentCompleted)

// WithPaymentID sets the payment ID for the PaymentCompletedEvent
func WithPaymentID(paymentID string) PaymentCompletedOpt {
	return func(e *PaymentCompleted) { e.PaymentID = paymentID }
}

// WithCorrelationID sets the correlation ID for the PaymentCompletedEvent
func WithCorrelationID(correlationID uuid.UUID) PaymentCompletedOpt {
	return func(e *PaymentCompleted) { e.CorrelationID = correlationID }
}

// NewPaymentCompleted creates a new PaymentCompleted with the given options
func NewPaymentCompleted(
	flowEvent FlowEvent,
	opts ...PaymentCompletedOpt,
) *PaymentCompleted {
	event := &PaymentCompleted{
		PaymentInitiated: PaymentInitiated{
			FlowEvent: flowEvent,
		},
	}

	for _, opt := range opts {
		opt(event)
	}

	return event
}
