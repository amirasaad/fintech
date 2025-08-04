package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/webhook"
)

// CheckoutSession represents a Stripe Checkout session.
type CheckoutSession struct {
	ID          string
	URL         string
	AmountTotal int64
	Currency    string
}

// PaymentEvent represents a payment event from Stripe.
type PaymentEvent struct {
	ID        string
	Status    provider.PaymentStatus
	Amount    int64
	Currency  string
	UserID    uuid.UUID
	AccountID uuid.UUID
}

// StripePaymentProvider implements PaymentProvider using Stripe API.
type StripePaymentProvider struct {
	client *stripe.Client
	cfg    *config.Stripe
	logger *slog.Logger
}

// NewStripePaymentProvider creates a new StripePaymentProvider with the given API key and logger.
func NewStripePaymentProvider(
	cfg *config.Stripe,
	logger *slog.Logger,
) *StripePaymentProvider {
	client := stripe.NewClient(cfg.ApiKey)
	return &StripePaymentProvider{client: client, cfg: cfg, logger: logger}
}

// InitiatePayment creates a PaymentIntent in Stripe and returns its ID.
func (s *StripePaymentProvider) InitiatePayment(
	ctx context.Context,
	userID, accountID uuid.UUID,
	amount int64,
	currency string,
) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
		Metadata: map[string]string{
			"user_id":    userID.String(),
			"account_id": accountID.String(),
		},
	}
	pi, err := paymentintent.New(params)
	log := s.logger.With(
		"handler", "stripe.InitiatePayment",
		"user_id", userID,
		"account_id", accountID,
		"amount", amount,
		"currency", currency,
	)
	if err != nil {
		log.Error(
			"[ERROR] stripe: failed to create payment intent",
			"err", err,
		)
		return "", fmt.Errorf("failed to create payment intent: %w", err)
	}
	return pi.ID, nil
}

// GetPaymentStatus retrieves the status of a PaymentIntent from Stripe.
func (s *StripePaymentProvider) GetPaymentStatus(
	ctx context.Context,
	paymentID string,
) (provider.PaymentStatus, error) {
	pi, err := s.client.V1PaymentIntents.Retrieve(ctx, paymentID, nil)
	log := s.logger.With(
		"handler", "stripe.GetPaymentStatus",
		"payment_id", paymentID,
	)
	if err != nil {
		log.Error(
			"[ERROR] stripe: failed to get payment intent",
			"err", err,
		)
		return provider.PaymentFailed, fmt.Errorf("failed to get payment intent: %w", err)
	}
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		return provider.PaymentCompleted, nil
	case stripe.PaymentIntentStatusProcessing,
		stripe.PaymentIntentStatusRequiresPaymentMethod,
		stripe.PaymentIntentStatusRequiresConfirmation:
		return provider.PaymentPending, nil
	default:
		return provider.PaymentFailed, nil
	}
}

// CreateCheckoutSession creates a new Stripe Checkout Session
func (s *StripePaymentProvider) CreateCheckoutSession(
	userID, accountID uuid.UUID,
	amount int64,
	currency string,
	successPath, cancelPath, description string,
) (*CheckoutSession, error) {
	successURL := s.ensureAbsoluteURL(successPath)
	cancelURL := s.ensureAbsoluteURL(cancelPath)

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(description),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(stripe.CheckoutSessionModePayment),
		SuccessURL: stripe.String(
			fmt.Sprintf("%s?session_id={CHECKOUT_SESSION_ID}",
				successURL)),
		CancelURL: stripe.String(cancelURL),
		Metadata: map[string]string{
			"user_id":    userID.String(),
			"account_id": accountID.String(),
		},
	}

	sess, err := session.New(params)
	if err != nil {
		s.logger.Error(
			"[ERROR] stripe: failed to create checkout session",
			"err", err,
		)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return &CheckoutSession{
		ID:          sess.ID,
		URL:         sess.URL,
		AmountTotal: sess.AmountTotal,
		Currency:    string(sess.Currency),
	}, nil
}

// HandleWebhook handles Stripe webhook events
func (s *StripePaymentProvider) HandleWebhook(
	payload []byte,
	signature string,
) (*PaymentEvent, error) {
	event, err := webhook.ConstructEvent(
		payload,
		signature,
		s.cfg.SigningSecret,
	)
	log := s.logger.With(
		"handler", "stripe.HandleWebhook",
		"event_type", event.Type)
	if err != nil {
		log.Error(
			"[ERROR] error verifying webhook signature",
			"err", err,
		)
		return nil, fmt.Errorf("error verifying webhook signature: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Error(
				"[ERROR] error parsing checkout.session.completed",
				"err", err,
			)
			return nil, fmt.Errorf("error parsing checkout.session.completed: %w", err)
		}

		// Get the payment intent details
		pi, err := paymentintent.Get(session.PaymentIntent.ID, nil)
		if err != nil {
			log.Error(
				"[ERROR] error retrieving payment intent",
				"err", err,
			)
			return nil, fmt.Errorf("error retrieving payment intent: %w", err)
		}

		userID, err := uuid.Parse(session.Metadata["user_id"])
		if err != nil {
			log.Error(
				"[ERROR] invalid user_id in metadata",
				"err", err,
			)
			return nil, fmt.Errorf("invalid user_id in metadata: %w", err)
		}

		accountID, err := uuid.Parse(session.Metadata["account_id"])
		if err != nil {
			log.Error(
				"[ERROR] invalid account_id in metadata",
				"err", err,
			)
			return nil, fmt.Errorf("invalid account_id in metadata: %w", err)
		}

		return &PaymentEvent{
			ID:        pi.ID,
			Status:    provider.PaymentCompleted,
			Amount:    pi.Amount,
			Currency:  string(pi.Currency),
			UserID:    userID,
			AccountID: accountID,
		}, nil

	case "payment_intent.succeeded":
		// Handle successful payment
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Error(
				"[ERROR] error parsing payment_intent.succeeded",
				"err", err,
			)
			return nil, fmt.Errorf("error parsing payment_intent.succeeded: %w", err)
		}

		log.Info(
			"Payment succeeded",
			"payment_intent_id", paymentIntent.ID,
		)

	default:
		log.Info(
			"Unhandled event type",
			"type", event.Type,
		)
	}

	return nil, nil
}

// ensureAbsoluteURL ensures the URL is absolute by prepending the base URL if needed
func (s *StripePaymentProvider) ensureAbsoluteURL(path string) string {
	if path == "" {
		return ""
	}

	u, err := url.Parse(path)
	if err != nil {
		return path
	}

	// If it's already an absolute URL, return as is
	if u.IsAbs() {
		return path
	}

	return path
}
