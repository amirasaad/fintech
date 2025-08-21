package account

import "github.com/amirasaad/fintech/pkg/money"

// FeeType represents the type of fee.
type FeeType string

const (
	// FeeProvider is the fee charged by Stripe.
	FeeProvider FeeType = "provider"
	// FeeTypeService is the platform's service fee.
	FeeTypeService FeeType = "service"
	// FeeTypeConversion is the fee for currency conversion.
	FeeTypeConversion FeeType = "conversion"
)

// Fee represents a fee associated with a transaction.
type Fee struct {
	Amount *money.Money
	Type   FeeType
}
