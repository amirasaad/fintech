package provider

import (
	"github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/amirasaad/fintech/pkg/provider/payment"
)

// Deprecated: Use exchange.Exchange instead.
type ExchangeRateProvider = exchange.Exchange

// Deprecated: Use payment.Payment instead.
type PaymentProvider = payment.Payment
