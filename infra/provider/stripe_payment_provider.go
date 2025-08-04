package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
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
	client          *stripe.Client
	cfg             *config.Stripe
	checkoutService *checkout.Service
	logger          *slog.Logger
}

// NewStripePaymentProvider creates a new StripePaymentProvider with the given API key and logger.
func NewStripePaymentProvider(
	cfg *config.Stripe,
	logger *slog.Logger,
) *StripePaymentProvider {
	client := stripe.NewClient(cfg.ApiKey)

	return &StripePaymentProvider{
		client: client,
		cfg:    cfg,
		checkoutService: checkout.NewService(
			registry.New(),
		),
		logger: logger,
	}
}

// InitiatePayment creates a PaymentIntent in Stripe and returns its ID.
func (s *StripePaymentProvider) InitiatePayment(
	ctx context.Context,
	params *provider.InitiatePaymentParams,
) (*provider.InitiatePaymentResponse, error) {
	log := s.logger.With(
		"handler", "stripe.InitiatePayment",
		"user_id", params.UserID,
		"account_id", params.AccountID,
		"amount", params.Amount,
		"currency", params.Currency,
	)
	co, err := s.createCheckoutSession(
		ctx,
		params.UserID,
		params.AccountID,
		params.Amount,
		params.Currency,
		"Payment for deposit",
	)
	if err != nil {
		log.Error(
			"[ERROR] failed to create checkout session",
			"err", err,
		)
		return nil,
			fmt.Errorf("failed to create checkout session: %w", err)

	}
	log.Info("created checkout session", "checkout_session", co)
	session, err := s.checkoutService.CreateSession(
		ctx,
		co.ID,
		params.TransactionID,
		params.UserID,
		params.AccountID,
		params.Amount,
		params.Currency,
		co.URL,
		time.Hour*24,
	)
	if err != nil {
		log.Error(
			"[ERROR] failed to create checkout session",
			"err", err,
		)
	}
	log.Info("created checkout session", "checkout_session", session)
	return &provider.InitiatePaymentResponse{
		Status: provider.PaymentPending,
	}, nil
}

// GetPaymentStatus retrieves the status of a PaymentIntent from Stripe.
func (s *StripePaymentProvider) GetPaymentStatus(
	ctx context.Context,
	params *provider.GetPaymentStatusParams,
) (provider.PaymentStatus, error) {
	pi, err := s.client.V1PaymentIntents.Retrieve(ctx, params.PaymentID, nil)
	log := s.logger.With(
		"handler", "stripe.GetPaymentStatus",
		"payment_id", params.PaymentID,
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

// createCheckoutSession creates a new Stripe Checkout Session
func (s *StripePaymentProvider) createCheckoutSession(
	ctx context.Context,
	userID, accountID uuid.UUID,
	amount int64,
	currency string,
	description string,
) (*CheckoutSession, error) {
	successURL := s.ensureAbsoluteURL(s.cfg.SuccessPath)
	cancelURL := s.ensureAbsoluteURL(s.cfg.CancelPath)

	params := &stripe.CheckoutSessionCreateParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String(successURL),
		CancelURL:          stripe.String(cancelURL),
		Metadata: map[string]string{
			"user_id":    userID.String(),
			"account_id": accountID.String(),
		},
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{{
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency: stripe.String(currency),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String(description)},
				UnitAmount: stripe.Int64(amount),
			},
			Quantity: stripe.Int64(1),
		}},
	}
	// Create the checkout session parameters

	// Add customer email if available
	if userEmail, ok := ctx.Value("user_email").(string); ok && userEmail != "" {
		params.CustomerEmail = stripe.String(userEmail)
	}

	// Create the checkout session using the session package
	session, err := s.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		s.logger.Error(
			"[ERROR] stripe: failed to create checkout session",
			"error", err,
		)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	// Log successful session creation
	s.logger.Info(
		"[INFO] stripe: created checkout session",
		"session_id", session.ID,
		"url", session.URL,
	)

	return &CheckoutSession{
		ID:          session.ID,
		URL:         session.URL,
		AmountTotal: session.AmountTotal,
		Currency:    string(session.Currency),
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
		pi, err := s.client.V1PaymentIntents.Retrieve(
			context.Background(),
			session.PaymentIntent.ID,
			nil,
		)
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
