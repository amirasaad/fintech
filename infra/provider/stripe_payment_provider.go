package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"time"

	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
)

// CheckoutSession represents a Stripe Checkout session.
type CheckoutSession struct {
	ID          string
	PaymentID   string
	URL         string
	AmountTotal int64
	Currency    string
}

// StripePaymentProvider implements PaymentProvider using Stripe API.
type StripePaymentProvider struct {
	bus             eventbus.Bus
	client          *stripe.Client
	cfg             *config.Stripe
	checkoutService *checkout.Service
	logger          *slog.Logger
}

// NewStripePaymentProvider creates a new StripePaymentProvider with the given
// API key, registry, and logger. The registry parameter is used for storing
// checkout session data.
func NewStripePaymentProvider(
	bus eventbus.Bus,
	cfg *config.Stripe,
	checkoutService *checkout.Service,
	logger *slog.Logger,
) *StripePaymentProvider {
	client := stripe.NewClient(cfg.ApiKey)

	return &StripePaymentProvider{
		bus:             bus,
		client:          client,
		cfg:             cfg,
		checkoutService: checkoutService,
		logger:          logger,
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

	// Create checkout session
	co, err := s.createCheckoutSession(
		ctx,
		params.UserID,
		params.AccountID,
		params.TransactionID,
		params.Amount,
		params.Currency,
		"Payment for deposit",
	)
	if err != nil {
		log.Error(
			"❌ Failed to create checkout session",
			"error", err,
		)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	// Create internal checkout session record
	_, err = s.checkoutService.CreateSession(
		ctx,
		co.ID,
		co.PaymentID,
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
			"❌ Failed to create checkout session record",
			"error", err,
		)
		return nil, fmt.Errorf(
			"failed to create checkout session record: %w", err)
	}

	log.Info(
		"🛒 Creating checkout session",
		"user_id", params.UserID,
		"account_id", params.AccountID,
		"transaction_id", params.TransactionID,
		"amount", params.Amount,
		"currency", params.Currency,
	)

	// Note: We're using the transaction ID as the payment ID to maintain consistency
	// with our internal transaction tracking. The checkout session ID is stored in the
	// checkout service for reference.
	return &provider.InitiatePaymentResponse{
		Status:    provider.PaymentPending,
		PaymentID: co.PaymentID, // Use transaction ID as payment ID
	}, nil
}

// HandleWebhook handles Stripe webhook events
func (s *StripePaymentProvider) HandleWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) (*provider.PaymentEvent, error) {
	// Verify the webhook signature
	// Create a logger with event context
	log := s.logger.With(
		"handler", "stripe.HandleWebhook",
	)
	event, err := webhook.ConstructEvent(
		payload,
		signature,
		s.cfg.SigningSecret,
	)

	if err != nil {
		log.Error(
			"❌ Invalid webhook signature",
			"error", err,
		)
		return nil, fmt.Errorf("error verifying webhook signature: %w", err)
	}
	log = log.With(
		"event_type", event.Type,
		"event_id", event.ID)
	log.Info(
		"📥 Handling webhook event",
		"type", event.Type,
	)

	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutSessionCompleted(ctx, event, log)

	case "checkout.session.expired":
		return s.handleCheckoutSessionExpired(ctx, event, log)

	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event, log)

	case "payment_intent.payment_failed":
		return s.handlePaymentIntentFailed(ctx, event, log)

	default:
		log.Info("Unhandled event type")
		return nil, nil
	}
}

func (s *StripePaymentProvider) GetPaymentStatus(
	ctx context.Context,
	params *provider.GetPaymentStatusParams,
) (provider.PaymentStatus, error) {
	pi, err := s.client.V1PaymentIntents.Retrieve(ctx, params.PaymentID, nil)
	log := s.logger.With(
		"handler", "stripe.GetPaymentStatus",
		"payment_id", params.PaymentID,
	)
	log.Info(
		"🔍 Getting payment status",
		"payment_id", params.PaymentID,
	)
	if err != nil {
		log.Error(
			"❌ Failed to get payment intent",
			"error", err,
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
	userID, accountID, transactionID uuid.UUID,
	amount int64,
	currency string,
	description string,
) (*CheckoutSession, error) {
	successURL := s.ensureAbsoluteURL(s.cfg.SuccessPath)
	cancelURL := s.ensureAbsoluteURL(s.cfg.CancelPath)

	// Create metadata for the checkout session and payment intent
	metadata := map[string]string{
		"user_id":        userID.String(),
		"account_id":     accountID.String(),
		"transaction_id": transactionID.String(),
		"amount":         fmt.Sprintf("%d", amount),
		"currency":       currency,
	}

	params := &stripe.CheckoutSessionCreateParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:         stripe.String(successURL),
		CancelURL:          stripe.String(cancelURL),
		Metadata:           metadata,
		PaymentIntentData: &stripe.CheckoutSessionCreatePaymentIntentDataParams{
			Metadata: metadata,
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
			"❌ stripe: failed to create checkout session",
			"error", err,
		)
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	// Log successful session creation
	s.logger.Info(
		"✅ Created checkout session",
		"session_id", session.ID,
		"url", session.URL,
	)

	// Create the checkout session response
	checkoutSession := &CheckoutSession{
		ID:          session.ID,
		URL:         session.URL,
		AmountTotal: session.AmountTotal,
		Currency:    string(session.Currency),
	}

	// Only set PaymentID if PaymentIntent is not nil
	if session.PaymentIntent != nil {
		checkoutSession.PaymentID = session.PaymentIntent.ID
	} else {
		// For some payment methods, the PaymentIntent might not be immediately available
		// In this case, we'll use the session ID as the payment ID
		checkoutSession.PaymentID = session.ID
	}

	return checkoutSession, nil
}

// handleCheckoutSessionCompleted handles the checkout.session.completed event
func (s *StripePaymentProvider) handleCheckoutSessionCompleted(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*provider.PaymentEvent, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Error(
			"❌ Error parsing checkout.session.completed",
			"error", err,
		)
		return nil, fmt.Errorf(
			"error parsing checkout.session.completed: %w", err)
	}

	log = log.With(
		"checkout_session_id", session.ID,
		"payment_intent_id", session.PaymentIntent.ID,
	)

	// Get transaction ID from metadata
	transactionID, err := uuid.Parse(session.Metadata["transaction_id"])
	if err != nil {
		log.Error(
			"❌ Invalid transaction_id in metadata",
			"error", err,
			"metadata", session.Metadata,
		)
		return nil, fmt.Errorf("invalid transaction_id in metadata: %w", err)
	}

	// Update the checkout session status with transaction context
	if err := s.checkoutService.UpdateStatus(
		context.Background(),
		session.ID,
		"completed",
	); err != nil {
		log.Error(
			"❌ Error updating checkout session status to completed",
			"error", err,
			"transaction_id", transactionID,
		)
		return nil, fmt.Errorf("error updating session status: %w", err)
	}

	// Get the payment intent details
	pi, err := s.client.V1PaymentIntents.Retrieve(
		context.Background(),
		session.PaymentIntent.ID,
		nil,
	)
	if err != nil {
		log.Error(
			"❌ Error retrieving payment intent",
			"error", err,
		)
		return nil, fmt.Errorf("error retrieving payment intent: %w", err)
	}

	// Get the user ID and account ID from metadata
	userID, err := uuid.Parse(session.Metadata["user_id"])
	if err != nil {
		log.Error(
			"❌ Invalid user_id in metadata",
			"error", err,
		)
		return nil, fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	accountID, err := uuid.Parse(session.Metadata["account_id"])
	if err != nil {
		log.Error(
			"❌ Invalid account_id in metadata",
			"error", err,
		)
		return nil, fmt.Errorf("invalid account_id in metadata: %w", err)
	}

	// Create metadata map from session metadata
	metadata := make(map[string]string)
	maps.Copy(metadata, session.Metadata)

	if err := s.bus.Emit(
		ctx,
		events.NewPaymentCompleted(
			events.FlowEvent{
				ID:     transactionID,
				UserID: userID,
			},
			events.WithPaymentID(pi.ID),
		),
	); err != nil {
		log.Error(
			"❌ Error emitting payment completed event",
			"error", err,
		)
		return nil, fmt.Errorf("error emitting payment completed event: %w", err)
	}

	log.Info(
		"✅ Checkout session and transaction updated successfully",
		"transaction_id", transactionID,
		"checkout_session_id", session.ID,
	)

	return &provider.PaymentEvent{
		ID:        pi.ID,
		Status:    provider.PaymentCompleted,
		Amount:    pi.Amount,
		Currency:  string(pi.Currency),
		UserID:    userID,
		AccountID: accountID,
		Metadata:  metadata,
	}, nil
}

// handleCheckoutSessionExpired handles the checkout.session.expired event
func (s *StripePaymentProvider) handleCheckoutSessionExpired(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*provider.PaymentEvent, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Error(
			"❌ Error parsing checkout.session.expired",
			"error", err,
		)
		return nil, fmt.Errorf(
			"error parsing checkout.session.expired: %w", err)
	}

	log = log.With(
		"checkout_session_id", session.ID,
		"payment_intent_id", session.PaymentIntent.ID,
	)

	// Get transaction ID from metadata
	transactionID, err := uuid.Parse(session.Metadata["transaction_id"])
	if err != nil {
		log.Error(
			"❌ Invalid transaction_id in metadata",
			"error", err,
			"metadata", session.Metadata,
		)
		return nil, fmt.Errorf("invalid transaction_id in metadata: %w", err)
	}

	// Update the checkout session status to expired
	if err := s.checkoutService.UpdateStatus(
		ctx,
		session.ID,
		"expired",
	); err != nil {
		log.Error(
			"❌ Error updating checkout session status to expired",
			"error", err,
			"transaction_id", transactionID,
		)
		return nil, fmt.Errorf("error updating session status: %w", err)
	}

	log.Info(
		"⏰ Checkout session and transaction updated to expired",
		"transaction_id", transactionID,
	)
	return nil, nil
}

// handlePaymentIntentSucceeded handles the payment_intent.succeeded event
func (s *StripePaymentProvider) handlePaymentIntentSucceeded(
	ctx context.Context,
	event stripe.Event, log *slog.Logger) (*provider.PaymentEvent, error) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Error(
			"❌ Error parsing payment_intent.succeeded",
			"error", err,
		)
		return nil, fmt.Errorf("error parsing payment_intent.succeeded: %w", err)
	}

	log = log.With("payment_intent_id", paymentIntent.ID)

	// Get the payment intent details
	pi, err := s.client.V1PaymentIntents.Retrieve(
		ctx,
		paymentIntent.ID,
		nil,
	)
	if err != nil {
		log.Error(
			"❌ Error retrieving payment intent",
			"error", err,
		)
		return nil, fmt.Errorf("error retrieving payment intent: %w", err)
	}

	// Get the user ID, account ID, and transaction ID from metadata
	userID, err := uuid.Parse(paymentIntent.Metadata["user_id"])
	if err != nil {
		log.Error(
			"❌ Invalid user_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	accountID, err := uuid.Parse(paymentIntent.Metadata["account_id"])
	if err != nil {
		log.Error(
			"❌ Invalid account_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid account_id in metadata: %w", err)
	}

	transactionID, err := uuid.Parse(
		paymentIntent.Metadata["transaction_id"])
	if err != nil {
		log.Error(
			"❌ Invalid transaction_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid transaction_id in metadata: %w", err)
	}

	// Create metadata map from payment intent metadata
	metadata := make(map[string]string)
	maps.Copy(metadata, paymentIntent.Metadata)

	log.Info(
		"💰 Handling payment_intent.succeeded event",
		"payment_intent_id", paymentIntent.ID,
	)

	// Update the transaction status using the transaction ID from metadata
	if err := s.bus.Emit(ctx, events.NewPaymentCompleted(
		events.FlowEvent{
			ID:            transactionID,
			UserID:        userID,
			AccountID:     accountID,
			FlowType:      "payment",
			CorrelationID: uuid.New(),
		},
		events.WithPaymentID(pi.ID),
	)); err != nil {
		log.Error(
			"❌ Error emitting payment completed event",
			"error", err,
		)
		return nil, fmt.Errorf("error emitting payment completed event: %w", err)
	}

	log.Info(
		"✅ Payment intent processed and transaction updated successfully",
		"transaction_id", transactionID,
		"payment_id", paymentIntent.ID,
	)

	return &provider.PaymentEvent{
		ID:        pi.ID,
		Status:    provider.PaymentCompleted,
		Amount:    pi.Amount,
		Currency:  string(pi.Currency),
		UserID:    userID,
		AccountID: accountID,
		Metadata:  metadata,
	}, nil
}

// handlePaymentIntentFailed handles the payment_intent.payment_failed event
func (s *StripePaymentProvider) handlePaymentIntentFailed(
	ctx context.Context,
	event stripe.Event, log *slog.Logger) (*provider.PaymentEvent, error) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Error(
			"❌ Error parsing payment_intent.payment_failed",
			"error", err,
		)
		return nil, fmt.Errorf(
			"error parsing payment_intent.payment_failed: %w", err)
	}

	log = log.With("payment_intent_id", paymentIntent.ID)

	// Get the payment intent details
	pi, err := s.client.V1PaymentIntents.Retrieve(
		context.Background(),
		paymentIntent.ID,
		nil,
	)
	if err != nil {
		log.Error(
			"❌ Error retrieving payment intent",
			"error", err,
		)
		return nil, fmt.Errorf("error retrieving payment intent: %w", err)
	}

	// Get the user ID, account ID, and transaction ID from metadata
	userID, err := uuid.Parse(paymentIntent.Metadata["user_id"])
	if err != nil {
		log.Error(
			"❌ Invalid user_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	accountID, err := uuid.Parse(paymentIntent.Metadata["account_id"])
	if err != nil {
		log.Error(
			"❌ Invalid account_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid account_id in metadata: %w", err)
	}

	transactionID, err := uuid.Parse(
		paymentIntent.Metadata["transaction_id"])
	if err != nil {
		log.Error(
			"❌ Invalid transaction_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid transaction_id in metadata: %w", err)
	}

	// Create metadata map from payment intent metadata
	metadata := make(map[string]string)
	maps.Copy(metadata, paymentIntent.Metadata)

	if err := s.bus.Emit(ctx, events.NewPaymentFailed(
		events.FlowEvent{
			ID:            transactionID,
			UserID:        userID,
			AccountID:     accountID,
			FlowType:      "payment",
			CorrelationID: uuid.New(),
		},
		events.WithFailedPaymentID(pi.ID),
	)); err != nil {
		log.Error(
			"❌ Error emitting payment failed event",
			"error", err,
		)
		return nil, fmt.Errorf("error emitting payment failed event: %w", err)
	}

	log.Info(
		"✅ Payment intent failed and transaction updated",
		"transaction_id", transactionID,
		"payment_id", paymentIntent.ID,
	)

	return &provider.PaymentEvent{
		ID:        pi.ID,
		Status:    provider.PaymentFailed,
		Amount:    pi.Amount,
		Currency:  string(pi.Currency),
		UserID:    userID,
		AccountID: accountID,
		Metadata:  metadata,
	}, nil
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
