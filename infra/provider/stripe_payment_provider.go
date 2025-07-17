package provider

import (
	"context"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/stripe/stripe-go/v82"
)

// StripePaymentProvider implements PaymentProvider using Stripe API.
type StripePaymentProvider struct {
	client *stripe.Client
	logger *slog.Logger
}

// NewStripePaymentProvider creates a new StripePaymentProvider with the given API key and logger.
func NewStripePaymentProvider(apiKey string, logger *slog.Logger) *StripePaymentProvider {
	client := stripe.NewClient(apiKey)
	return &StripePaymentProvider{client: client, logger: logger}
}

// InitiatePayment creates a PaymentIntent in Stripe and returns its ID.
func (s *StripePaymentProvider) InitiatePayment(ctx context.Context, userID, accountID string, amount int64, currency string) (string, error) {
	params := &stripe.PaymentIntentCreateParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
		Metadata: map[string]string{
			"user_id":    userID,
			"account_id": accountID,
		},
	}
	pi, err := s.client.V1PaymentIntents.Create(ctx, params)
	if err != nil {
		s.logger.Error("stripe: failed to create payment intent", "err", err)
		return "", err
	}
	return pi.ID, nil
}

// GetPaymentStatus retrieves the status of a PaymentIntent from Stripe.
func (s *StripePaymentProvider) GetPaymentStatus(ctx context.Context, paymentID string) (provider.PaymentStatus, error) {
	pi, err := s.client.V1PaymentIntents.Retrieve(ctx, paymentID, nil)
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
