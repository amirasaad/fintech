package payment

import (
	"context"
)

// Payment is a interface for payment provider
type Payment interface {
	InitiatePayment(
		ctx context.Context,
		params *InitiatePaymentParams,
	) (*InitiatePaymentResponse, error)
	HandleWebhook(
		ctx context.Context,
		payload []byte,
		signature string,
	) (*PaymentEvent, error)
}
