package events

import (
	"time"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
)

// FeesCalculatedOpt is a function that configures a FeesCalculated event
type FeesCalculatedOpt func(*FeesCalculated)

// WithFeeTransactionID sets the transaction ID for the FeesCalculated event
func WithFeeTransactionID(id uuid.UUID) FeesCalculatedOpt {
	return func(e *FeesCalculated) { e.TransactionID = id }
}

// WithFee sets the fee amount for the FeesCalculated event
func WithFee(fee account.Fee) FeesCalculatedOpt {
	return func(e *FeesCalculated) { e.Fee = fee }
}

// NewFeesCalculated creates a new FeesCalculated event with the given options
func NewFeesCalculated(ef *FlowEvent, opts ...FeesCalculatedOpt) *FeesCalculated {
	e := &FeesCalculated{
		FlowEvent: *ef,
	}

	// Set default values
	e.ID = uuid.New()
	e.Timestamp = time.Now()

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// WithFeeType sets the fee type for the FeesCalculated event
func WithFeeType(feeType account.FeeType) FeesCalculatedOpt {
	return func(e *FeesCalculated) {
		e.Fee.Type = feeType
	}
}

// WithFeeAmountValue sets the fee amount for the FeesCalculated event
func WithFeeAmountValue(amount *money.Money) FeesCalculatedOpt {
	return func(e *FeesCalculated) {
		e.Fee.Amount = amount
	}
}
