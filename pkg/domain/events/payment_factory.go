package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
)

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
func WithFlowEvent(fe FlowEvent) PaymentInitiatedOpt {
	return func(e *PaymentInitiated) {
		e.FlowEvent = fe
	}
}

// NewPaymentInitiated creates a new PaymentInitiated with the given options
func NewPaymentInitiated(fe FlowEvent, opts ...PaymentInitiatedOpt) *PaymentInitiated {
	pi := &PaymentInitiated{
		FlowEvent:     fe,
		TransactionID: uuid.Nil,
		PaymentID:     "",
		Status:        "initiated",
	}

	pi.ID = uuid.New()
	pi.Timestamp = time.Now()
	for _, opt := range opts {
		opt(pi)
	}

	return pi
}

type PaymentProcessedOpt func(*PaymentProcessed)

// NewPaymentProcessed creates a new PaymentProcessed with the given parameters
func NewPaymentProcessed(
	pi PaymentInitiated,
	opts ...PaymentProcessedOpt,
) *PaymentProcessed {
	// Create base PaymentInitiated with required fields
	pp := &PaymentProcessed{
		PaymentInitiated: pi,
	}

	pp.ID = uuid.New()
	pp.Timestamp = time.Now()
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

// WithProviderFee sets the provider fee for the PaymentCompletedEvent
func WithProviderFee(providerFee account.Fee) PaymentCompletedOpt {
	return func(e *PaymentCompleted) { e.ProviderFee = providerFee }
}

// WithCorrelationID sets the correlation ID for the PaymentCompletedEvent
func WithCorrelationID(correlationID uuid.UUID) PaymentCompletedOpt {
	return func(e *PaymentCompleted) { e.CorrelationID = correlationID }
}

// NewPaymentCompleted creates a new PaymentCompleted with the given options
func NewPaymentCompleted(
	fe FlowEvent,
	opts ...PaymentCompletedOpt,
) *PaymentCompleted {
	pc := &PaymentCompleted{
		PaymentInitiated: PaymentInitiated{
			FlowEvent: fe,
		},
	}

	pc.ID = uuid.New()
	pc.Timestamp = time.Now()
	for _, opt := range opts {
		opt(pc)
	}

	return pc
}

// PaymentFailedOpt is a function that configures a PaymentFailedEvent
type PaymentFailedOpt func(*PaymentFailed)

// WithPaymentID sets the payment ID for the PaymentFailedEvent
func WithFailedPaymentID(paymentID string) PaymentFailedOpt {
	return func(e *PaymentFailed) { e.PaymentID = paymentID }
}

// NewPaymentFailed creates a new PaymentFailed with the given options
func NewPaymentFailed(
	fe FlowEvent,
	opts ...PaymentFailedOpt,
) *PaymentFailed {
	pf := &PaymentFailed{
		PaymentInitiated: PaymentInitiated{
			FlowEvent: fe,
		},
	}

	pf.ID = uuid.New()
	pf.Timestamp = time.Now()
	for _, opt := range opts {
		opt(pf)
	}

	return pf
}
