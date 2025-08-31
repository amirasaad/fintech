package provider

import (
	"github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/amirasaad/fintech/pkg/provider/payment"
)

// Exchange
type (
	RateInfo         = exchange.RateInfo
	RateFetcher      = exchange.RateFetcher
	HealthChecker    = exchange.HealthChecker
	SupportedChecker = exchange.SupportedChecker
	ProviderMetadata = exchange.ProviderMetadata
	Provider         = exchange.Exchange
)

// Payment
type (
	PaymentStatus             = payment.PaymentStatus
	PaymentEvent              = payment.PaymentEvent
	InitiatePaymentParams     = payment.InitiatePaymentParams
	InitiatePaymentResponse   = payment.InitiatePaymentResponse
	GetPaymentStatusParams    = payment.GetPaymentStatusParams
	UpdatePaymentStatusParams = payment.UpdatePaymentStatusParams
	Payment                   = payment.Payment
)
