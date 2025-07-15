package provider

import (
	"context"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

// StripePaymentProvider implements PaymentProvider using Stripe API.
type StripePaymentProvider struct {
	apiKey string
	logger *slog.Logger
}

// NewStripePaymentProvider creates a new StripePaymentProvider with the given API key and logger.
func NewStripePaymentProvider(apiKey string, logger *slog.Logger) *StripePaymentProvider {
	stripe.Key = apiKey
	return &StripePaymentProvider{apiKey: apiKey, logger: logger}
}

// InitiatePayment creates a PaymentIntent in Stripe and returns its ID.
func (s *StripePaymentProvider) InitiatePayment(ctx context.Context, userID, accountID uuid.UUID, amount float64, currency string) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount * 100)), // Stripe expects amount in the smallest currency unit
		Currency: stripe.String(currency),
		Metadata: map[string]string{
			"user_id":    userID.String(),
			"account_id": accountID.String(),
		},
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		s.logger.Error("stripe: failed to create payment intent", "err", err)
		return "", err
	}
	return pi.ID, nil
}

// GetPaymentStatus retrieves the status of a PaymentIntent from Stripe.
func (s *StripePaymentProvider) GetPaymentStatus(ctx context.Context, paymentID string) (provider.PaymentStatus, error) {
	pi, err := paymentintent.Get(paymentID, nil)
	if err != nil {
		s.logger.Error("stripe: failed to get payment intent", "err", err)
		return provider.PaymentFailed, err
	}
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		return provider.PaymentCompleted, nil
	case stripe.PaymentIntentStatusProcessing, stripe.PaymentIntentStatusRequiresPaymentMethod, stripe.PaymentIntentStatusRequiresConfirmation:
		return provider.PaymentPending, nil
	default:
		return provider.PaymentFailed, nil
	}
}
