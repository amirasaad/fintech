package stripepayment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/amirasaad/fintech/pkg/service/checkout"

	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/repository"

	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/payment"
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

// StripePaymentProvider implements Payment using Stripe API.
type StripePaymentProvider struct {
	bus             eventbus.Bus
	client          *stripe.Client
	checkoutService *checkout.Service
	cfg             *config.Stripe
	logger          *slog.Logger
	webhookHandlers map[string]webhookHandler
	uow             repository.UnitOfWork
}

type webhookHandler func(context.Context, stripe.Event, *slog.Logger) (*payment.PaymentEvent, error)

// New creates a new StripePaymentProvider with the given
// API key, registry, and logger. The registry parameter is used for storing
// checkout session data.
func New(
	bus eventbus.Bus,
	checkoutProvider registry.Provider,
	cfg *config.Stripe,
	logger *slog.Logger,
	uow repository.UnitOfWork,
) *StripePaymentProvider {
	client := stripe.NewClient(cfg.ApiKey)

	provider := &StripePaymentProvider{
		bus:             bus,
		client:          client,
		cfg:             cfg,
		checkoutService: checkout.New(checkoutProvider),
		logger:          logger,
		webhookHandlers: make(map[string]webhookHandler),
		uow:             uow,
	}

	// Initialize webhook handlers
	provider.initializeWebhookHandlers()

	return provider
}

// initializeWebhookHandlers sets up all the webhook handlers for Stripe events
func (s *StripePaymentProvider) initializeWebhookHandlers() {
	s.webhookHandlers = make(map[string]webhookHandler)

	// Payment intent events
	s.webhookHandlers["payment_intent.succeeded"] = s.handlePaymentIntentSucceeded
	s.webhookHandlers["payment_intent.payment_failed"] = s.handlePaymentIntentFailed

	// Checkout session events
	s.webhookHandlers["checkout.session.completed"] = s.handleCheckoutSessionCompleted
	s.webhookHandlers["checkout.session.expired"] = s.handleCheckoutSessionExpired

	// Transfer events
	s.webhookHandlers["transfer.created"] = s.handleTransferCreated
	s.webhookHandlers["transfer.failed"] = s.handleTransferFailed
	s.webhookHandlers["transfer.reversed"] = s.handleTransferReversed

	// Charge events
	s.webhookHandlers["charge.succeeded"] = s.handleChargeSucceeded
	s.webhookHandlers["charge.updated"] = s.handleChargeSucceeded

	// Account events
	s.webhookHandlers["account.updated"] = s.handleAccountUpdated
	s.webhookHandlers["account.application.authorized"] = s.handleAccountApplicationAuthorized
	s.webhookHandlers["capability.updated"] = s.handleCapabilityUpdated

	// Payout events
	// s.webhookHandlers["payout.paid"] = s.handlePayoutPaid
	// s.webhookHandlers["payout.failed"] = s.handlePayoutFailed
}

func (s *StripePaymentProvider) handleAccountUpdated(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var account stripe.Account
	if err := json.Unmarshal(event.Data.Raw, &account); err != nil {
		return nil, fmt.Errorf("error parsing account: %v", err)
	}

	log.Info("Account updated",
		"account_id", account.ID,
		"details_submitted", account.DetailsSubmitted,
	)

	if account.DetailsSubmitted {
		userID, err := uuid.Parse(account.Metadata["user_id"])
		if err != nil {
			return nil, fmt.Errorf("error parsing user_id from account metadata: %v", err)
		}

		// Emit a custom event to notify the system that the user has completed onboarding.
		onboardingCompletedEvent := events.NewUserOnboardingCompleted(userID, account.ID)
		if err := s.bus.Emit(ctx, onboardingCompletedEvent); err != nil {
			log.Error("failed to emit UserOnboardingCompleted event", "error", err)
			return nil, fmt.Errorf("failed to emit UserOnboardingCompleted event: %w", err)
		}
	}

	return nil, nil
}

func (s *StripePaymentProvider) handleAccountApplicationAuthorized(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var app stripe.Application
	if err := json.Unmarshal(event.Data.Raw, &app); err != nil {
		return nil, fmt.Errorf("error parsing application: %v", err)
	}

	log.Info("Account application authorized",
		"application_id", app.ID,
	)

	// This event indicates that the user has authorized the application
	// to connect to their Stripe account.
	// We can treat this as the user having completed the onboarding process.
	userID, err := uuid.Parse(event.Account)
	if err != nil {
		return nil, fmt.Errorf("error parsing user_id from event account: %v", err)
	}

	onboardingCompletedEvent := events.NewUserOnboardingCompleted(userID, event.Account)
	if err := s.bus.Emit(ctx, onboardingCompletedEvent); err != nil {
		log.Error("failed to emit UserOnboardingCompleted event", "error", err)
		return nil, fmt.Errorf("failed to emit UserOnboardingCompleted event: %w", err)
	}

	return nil, nil
}

func (s *StripePaymentProvider) handleCapabilityUpdated(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var capability stripe.Capability
	if err := json.Unmarshal(event.Data.Raw, &capability); err != nil {
		return nil, fmt.Errorf("error parsing capability: %v", err)
	}

	log.Info("Capability updated",
		"capability_id", capability.ID,
		"status", capability.Status,
		"account", capability.Account.ID,
	)

	if capability.ID == "transfers" && capability.Status == stripe.CapabilityStatusActive {
		userID, err := uuid.Parse(capability.Account.Metadata["user_id"])
		if err != nil {
			return nil, fmt.Errorf("error parsing user_id from account metadata: %v", err)
		}

		// Emit a custom event to notify the system that the user has completed onboarding.
		onboardingCompletedEvent := events.NewUserOnboardingCompleted(userID, capability.Account.ID)
		if err := s.bus.Emit(ctx, onboardingCompletedEvent); err != nil {
			log.Error("failed to emit UserOnboardingCompleted event", "error", err)
			return nil, fmt.Errorf("failed to emit UserOnboardingCompleted event: %w", err)
		}
	}

	return nil, nil
}

// InitiatePayment creates a PaymentIntent in Stripe and returns its ID.
func (s *StripePaymentProvider) InitiatePayment(
	ctx context.Context,
	params *payment.InitiatePaymentParams,
) (*payment.InitiatePaymentResponse, error) {
	s.logger.Debug("🔵 InitiatePayment called",
		"transaction_id", params.TransactionID,
		"amount", params.Amount,
		"currency", params.Currency,
	)
	log := s.logger.With(
		"handler", "stripe.InitiatePayment",
		"user_id", params.UserID,
		"account_id", params.AccountID,
		"amount", params.Amount,
		"currency", params.Currency,
	)
	log.Info("🛒 [START] InitiatePayment")

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
			"failed to create checkout session",
			"error", err,
		)
		return nil, fmt.Errorf(
			"failed to create checkout session: %w", err)
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
			"failed to create checkout session record",
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

	return &payment.InitiatePaymentResponse{
		Status:    payment.PaymentPending,
		PaymentID: co.PaymentID,
	}, nil
}

// VerifyWebhookSignature verifies the signature of a webhook event
func (s *StripePaymentProvider) VerifyWebhookSignature(payload []byte, header string) error {
	if s.cfg.SigningSecret == "" {
		return fmt.Errorf("webhook signing secret not configured")
	}

	_, err := webhook.ConstructEvent(payload, header, s.cfg.SigningSecret)
	if err != nil {
		return fmt.Errorf("error verifying webhook signature: %v", err)
	}

	s.logger.Info("Webhook signature verified", "signature", header)
	return nil
}

// HandleWebhook handles incoming webhook events from Stripe
func (s *StripePaymentProvider) HandleWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) (*payment.PaymentEvent, error) {
	log := s.logger.With("method", "HandleWebhook")

	// Verify the webhook signature
	if err := s.VerifyWebhookSignature(payload, signature); err != nil {
		log.Error("Failed to verify webhook signature", "error", err)
		return nil, fmt.Errorf("webhook signature verification failed: %v", err)
	}

	// Parse the webhook event
	event := stripe.Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Error("Failed to parse webhook event", "error", err)
		return nil, fmt.Errorf("error parsing webhook event: %v", err)
	}

	log.Info("Received webhook event",
		"type", event.Type,
		"id", event.ID,
	)

	// Find the appropriate handler for the event type
	handler, ok := s.webhookHandlers[string(event.Type)]
	if !ok {
		log.Warn("No handler found for event type", "type", event.Type)
		return nil, fmt.Errorf("unhandled event type: %s", event.Type)
	}

	return handler(ctx, event, log)
}

// handleTransferCreated handles transfer.created webhook events
func (s *StripePaymentProvider) handleTransferCreated(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	log.Debug("🔵 handleTransferCreated called",
		"event_id", event.ID,
		"event_type", event.Type,
		"event_data", string(event.Data.Raw),
	)
	var transfer stripe.Transfer
	if err := json.Unmarshal(event.Data.Raw, &transfer); err != nil {
		return nil, fmt.Errorf("error parsing transfer: %v", err)
	}

	log.Info("Transfer created",
		"transfer_id", transfer.ID,
		"amount", transfer.Amount,
		"currency", transfer.Currency,
	)

	// Get metadata safely
	metadata := make(map[string]string)
	if transfer.Metadata != nil {
		metadata = transfer.Metadata
	}

	// Parse user and account IDs from metadata
	userID, _ := uuid.Parse(metadata["user_id"])
	accountID, _ := uuid.Parse(metadata["account_id"])
	transactionID, _ := uuid.Parse(metadata["transaction_id"])

	// Create money amount - convert from cents to dollars for money package
	amount, err := s.parseAmount(transfer.Amount, string(transfer.Currency))
	if err != nil {
		return nil, fmt.Errorf("error creating money amount: %v", err)
	}

	// Determine payment status based on transfer status
	status := payment.PaymentCompleted
	if transfer.Reversed {
		status = payment.PaymentFailed
	} else if transfer.AmountReversed > 0 {
		status = payment.PaymentStatus("partially_reversed")
	}

	// Build the payment completed event
	metadataInfo := &metadataInfo{
		UserID:        userID,
		AccountID:     accountID,
		TransactionID: transactionID,
		PaymentID:     transfer.ID,
	}

	// Create the payment completed event
	pc := s.buildPaymentCompletedEventPayload(amount.Negate(), transfer.ID, metadataInfo, log)
	if pc == nil {
		return nil, fmt.Errorf("failed to build payment completed event payload")
	}

	// Log the event details
	log.Debug("🔵 Emitting PaymentCompleted event",
		"event_id", pc.ID,
		"transaction_id", transactionID,
		"transfer_id", transfer.ID,
		"amount", amount.Amount(),
		"currency", amount.Currency(),
	)

	// Emit the event
	if err := s.bus.Emit(ctx, pc); err != nil {
		log.Error("🔴 Failed to emit PaymentCompleted event",
			"error", err,
			"event_id", pc.ID,
		)
		return nil, fmt.Errorf("failed to emit payment completed event: %v", err)
	}

	log.Info("✅ PaymentCompleted event emitted successfully",
		"event_id", pc.ID,
		"transaction_id", transactionID,
		"transfer_id", transfer.ID,
	)

	payoutEvent := &payment.PaymentEvent{
		ID:            transfer.ID,
		Status:        status,
		Amount:        amount.Amount(),
		UserID:        userID,
		AccountID:     accountID,
		TransactionID: transactionID,
		Metadata:      metadata,
	}

	return payoutEvent, nil
}

// handleTransferFailed handles transfer.failed webhook events
func (s *StripePaymentProvider) handleTransferFailed(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var transfer stripe.Transfer
	if err := json.Unmarshal(event.Data.Raw, &transfer); err != nil {
		return nil, fmt.Errorf("error parsing transfer: %v", err)
	}

	// Get metadata safely
	metadata := make(map[string]string)
	if transfer.Metadata != nil {
		metadata = transfer.Metadata
	}

	// Parse user and account IDs from metadata
	userID, _ := uuid.Parse(metadata["user_id"])
	accountID, _ := uuid.Parse(metadata["account_id"])

	// Get failure reason from metadata or use a default message
	failureReason := metadata["failure_reason"]
	if failureReason == "" {
		failureReason = "transfer failed"
	}

	log.Error("Transfer failed",
		"transfer_id", transfer.ID,
		"amount", transfer.Amount,
		"currency", transfer.Currency,
		"failure_reason", failureReason,
	)

	// Create money amount - convert from cents to dollars for money package
	amount, err := s.parseAmount(transfer.Amount, string(transfer.Currency))
	if err != nil {
		return nil, fmt.Errorf("error creating money amount: %v", err)
	}

	// Try to get transaction ID from metadata if available
	transactionID := uuid.Nil
	if txID, ok := metadata["transaction_id"]; ok && txID != "" {
		transactionID, _ = uuid.Parse(txID)
	}

	payoutEvent := &payment.PaymentEvent{
		ID:            transfer.ID,
		Status:        payment.PaymentFailed,
		Amount:        amount.Amount(),
		UserID:        userID,
		AccountID:     accountID,
		TransactionID: transactionID,
		Metadata:      metadata,
	}

	return payoutEvent, nil
}

// handleTransferReversed handles transfer.reversed webhook events
func (s *StripePaymentProvider) handleTransferReversed(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var transfer stripe.Transfer
	if err := json.Unmarshal(event.Data.Raw, &transfer); err != nil {
		return nil, fmt.Errorf("error parsing transfer: %v", err)
	}

	log.Warn("Transfer reversed",
		"transfer_id", transfer.ID,
		"amount", transfer.Amount,
		"currency", transfer.Currency,
	)

	// Get metadata safely
	metadata := make(map[string]string)
	if transfer.Metadata != nil {
		metadata = transfer.Metadata
	}

	// Parse user and account IDs from metadata
	userID, _ := uuid.Parse(metadata["user_id"])
	accountID, _ := uuid.Parse(metadata["account_id"])

	// Create money amount - convert from cents to dollars for money package
	amount, err := s.parseAmount(transfer.Amount, string(transfer.Currency))
	if err != nil {
		return nil, fmt.Errorf("error creating money amount: %v", err)
	}

	// Try to get transaction ID from metadata if available
	transactionID := uuid.Nil
	if txID, ok := metadata["transaction_id"]; ok && txID != "" {
		transactionID, _ = uuid.Parse(txID)
	}

	// Create a failure reason based on whether it's a full or partial reversal
	failureReason := "transfer fully reversed"
	if transfer.AmountReversed > 0 && transfer.AmountReversed < transfer.Amount {
		failureReason = fmt.Sprintf(
			"transfer partially reversed: %d/%d",
			transfer.AmountReversed, transfer.Amount,
		)
	}

	payoutEvent := &payment.PaymentEvent{
		ID:            transfer.ID,
		Status:        payment.PaymentFailed, // Using PaymentFailed since there's no specific err
		Amount:        amount.Amount(),
		UserID:        userID,
		AccountID:     accountID,
		TransactionID: transactionID,
		Metadata:      metadata,
	}

	// Add failure reason to metadata for reference
	if payoutEvent.Metadata == nil {
		payoutEvent.Metadata = make(map[string]string)
	}
	payoutEvent.Metadata["failure_reason"] = failureReason

	return payoutEvent, nil
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
			"failed to create checkout session",
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
	}

	return checkoutSession, nil
}

// handleCheckoutSessionCompleted handles the checkout.session.completed event
func (s *StripePaymentProvider) handleCheckoutSessionCompleted(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Error(
			"parsing checkout.session.completed",
			"error", err,
		)
		return nil, fmt.Errorf(
			"error parsing checkout.pi.completed: %w", err)
	}

	log = log.With(
		"checkout_session_id", session.ID,
		"payment_intent_id", session.PaymentIntent.ID,
	)
	se, err := s.checkoutService.GetSession(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	amount, err := s.parseAmount(session.AmountSubtotal, string(session.Currency))
	if err != nil {
		log.Error(
			"error parsing amount",
			"error", err,
		)
		return nil, fmt.Errorf("error parsing amount: %w", err)
	}

	if err := s.bus.Emit(
		ctx,
		events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            uuid.New(),
				UserID:        se.UserID,
				AccountID:     se.AccountID,
				FlowType:      "payment",
				CorrelationID: uuid.New(),
			}, func(pp *events.PaymentProcessed) {
				pp.TransactionID = se.TransactionID
				paymentID := session.PaymentIntent.ID
				pp.PaymentID = &paymentID
				log.Info("Emitting ", "event_type", pp.Type())
			},
		).WithAmount(amount).WithPaymentID(session.PaymentIntent.ID),
	); err != nil {
		log.Error(
			"error emitting payment processed event",
			"error", err,
		)
		return nil, fmt.Errorf("error emitting payment processed event: %w", err)
	}

	log.Info(
		"✅ Checkout pi and transaction updated successfully",
		"transaction_id", se.TransactionID,
		"checkout_session_id", session.ID,
		"payment_intent_id", session.PaymentIntent.ID,
	)

	return &payment.PaymentEvent{
		ID:        session.PaymentIntent.ID,
		Status:    payment.PaymentCompleted,
		Amount:    session.PaymentIntent.AmountReceived,
		Currency:  string(session.Currency),
		UserID:    se.UserID,
		AccountID: se.AccountID,
	}, nil
}

// handleCheckoutSessionExpired handles the checkout.session.expired event
func (s *StripePaymentProvider) handleCheckoutSessionExpired(
	ctx context.Context,
	event stripe.Event,
	log *slog.Logger,
) (*payment.PaymentEvent, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Error(
			"parsing checkout.session.expired",
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
			"invalid transaction_id in metadata",
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
			"updating checkout session status to expired",
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
	event stripe.Event,
	log *slog.Logger,
) (
	*payment.PaymentEvent,
	error,
) {
	const op = "stripe.handlePaymentIntentSucceeded"

	if event.Data == nil || event.Data.Raw == nil {
		err := fmt.Errorf("%s: event data is nil", op)
		log.Error(err.Error())
		return nil, err
	}

	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		err = fmt.Errorf("%s: failed to unmarshal payment intent: %w", op, err)
		log.Error(err.Error())
		return nil, err
	}

	if pi.ID == "" {
		err := fmt.Errorf("%s: payment intent ID is empty", op)
		log.Error(err.Error())
		return nil, err
	}
	log = log.With("payment_intent_id", pi.ID)

	log.Info("💰 Handling payment_intent.succeeded event", "payment_intent_id", pi.ID)
	if pi.Metadata == nil {
		err := fmt.Errorf("%s: payment intent metadata is nil", op)
		log.Error(err.Error())
		return nil, err
	}

	parsedMeta, err := s.parseAndValidateMetadata(pi.Metadata, log)
	if err != nil {
		err = fmt.Errorf("%s: invalid metadata: %w", op, err)
		log.Error(err.Error())
		return nil, err
	}
	metadata := s.copyMetadata(pi.Metadata)

	currencyCode := strings.ToUpper(string(pi.Currency))
	if currencyCode == "" {
		err = fmt.Errorf("%s: currency code is empty", op)
		log.Error(err.Error())
		return nil, err
	}
	amount, err := s.parseAmount(pi.AmountReceived, currencyCode)
	if err != nil {
		log.Error("failed to create money amount",
			"error", err,
			"amount", pi.AmountReceived,
			"currency", currencyCode)
		return nil, fmt.Errorf("failed to create money amount: %w", err)
	}
	// Emit PaymentCompleted event with zero fee since we're dropping fees
	pc := s.buildPaymentCompletedEventPayload(amount, pi.ID, parsedMeta, log)
	if pc == nil {
		err := fmt.Errorf("failed to build payment completed event payload")
		log.Error(err.Error())
		return nil, err
	}
	if err := s.bus.Emit(ctx, pc); err != nil {
		log.Error("error emitting payment completed event", "error", err)
		return nil, fmt.Errorf("error emitting payment completed event: %w", err)
	}

	log.Info("✅ Payment intent processed and transaction updated successfully",
		"transaction_id", parsedMeta.TransactionID, "payment_id", pi.ID)
	return &payment.PaymentEvent{
		ID:        pi.ID,
		Status:    payment.PaymentCompleted,
		Amount:    pi.AmountReceived,
		Currency:  string(pi.Currency),
		UserID:    parsedMeta.UserID,
		AccountID: parsedMeta.AccountID,
		Metadata:  metadata,
	}, nil
}

// getFeeFromBalanceTransaction retrieves the balance transaction
// and returns the fee amount and currency.
func (s *StripePaymentProvider) getFeeFromBalanceTransaction(
	ctx context.Context,
	log *slog.Logger,
	balanceTxID string,
) (int64, string, error) {
	bt, err := s.client.V1BalanceTransactions.Retrieve(ctx, balanceTxID, nil)
	if err != nil {
		log.Warn(
			"Failed to retrieve balance transaction",
			"error", err,
			"balance_transaction_id", balanceTxID,
		)
		return 0, "", err
	}
	log.Debug("Retrieved balance transaction", "balance_transaction", bt)
	feeAmount := bt.Fee
	feeCurrency := strings.ToUpper(string(bt.Currency))
	log.Info("Retrieved fee from balance transaction",
		"fee_amount", feeAmount,
		"fee_currency", feeCurrency,
		"balance_transaction_id", balanceTxID,
	)
	return feeAmount, feeCurrency, nil
}

// metadataInfo holds parsed metadata fields.
type metadataInfo struct {
	UserID        uuid.UUID
	AccountID     uuid.UUID
	TransactionID uuid.UUID
	PaymentID     string
	Currency      string
}

// parseAndValidateMetadata extracts and validates required fields from metadata.
func (s *StripePaymentProvider) parseAndValidateMetadata(
	meta map[string]string,
	log *slog.Logger,
) (*metadataInfo, error) {
	const op = "stripe.parseAndValidateMetadata"

	// Check for required fields
	requiredFields := []string{"user_id", "account_id", "transaction_id", "currency"}
	var missingFields []string

	for _, field := range requiredFields {
		if _, exists := meta[field]; !exists || meta[field] == "" {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		err := fmt.Errorf("%s: missing required metadata fields: %v", op, missingFields)
		log.Error(err.Error(), "metadata", meta)
		return nil, err
	}

	// Parse UUIDs
	userID, err := uuid.Parse(meta["user_id"])
	if err != nil {
		err = fmt.Errorf("%s: invalid user_id in metadata: %w", op, err)
		log.Error(err.Error(), "user_id", meta["user_id"])
		return nil, err
	}

	accountID, err := uuid.Parse(meta["account_id"])
	if err != nil {
		err = fmt.Errorf("%s: invalid account_id in metadata: %w", op, err)
		log.Error(err.Error(), "account_id", meta["account_id"])
		return nil, err
	}

	transactionID, err := uuid.Parse(meta["transaction_id"])
	if err != nil {
		err = fmt.Errorf("%s: invalid transaction_id in metadata: %w", op, err)
		log.Error(err.Error(), "transaction_id", meta["transaction_id"])
		return nil, err
	}

	// Validate currency
	currencyCode := strings.TrimSpace(meta["currency"])
	if currencyCode == "" {
		err := fmt.Errorf("%s: currency code is empty", op)
		log.Error(err.Error())
		return nil, err
	}

	// Convert to uppercase for consistency
	currencyCode = strings.ToUpper(currencyCode)

	// Basic currency code validation (ISO 4217 format - 3 uppercase letters)
	if len(currencyCode) != 3 || !isAlpha(currencyCode) {
		err := fmt.Errorf("%s: invalid currency code format: %s", op, currencyCode)
		log.Error(err.Error())
		return nil, err
	}

	return &metadataInfo{
		UserID:        userID,
		AccountID:     accountID,
		TransactionID: transactionID,
		Currency:      currencyCode,
	}, nil
}

// copyMetadata creates a copy of the metadata map.
func (s *StripePaymentProvider) copyMetadata(
	meta map[string]string,
) map[string]string {
	if meta == nil {
		return make(map[string]string)
	}

	copied := make(map[string]string, len(meta))
	for k, v := range meta {
		if k != "" {
			copied[k] = v
		}
	}
	return copied
}

// isAlpha checks if a string contains only letters.
func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// parseAmount converts a Stripe amount and currency to a money.Money object.
// It validates the currency code and ensures the amount is non-negative.
func (s *StripePaymentProvider) parseAmount(
	amount int64,
	currency string,
) (*money.Money, error) {
	const op = "stripe.parseAmount"

	if amount < 0 {
		err := fmt.Errorf("%s: amount cannot be negative: %d", op, amount)
		s.logger.Error(err.Error())
		return nil, err
	}

	if currency == "" {
		err := fmt.Errorf("%s: currency cannot be empty", op)
		s.logger.Error(err.Error())
		return nil, err
	}

	// Convert to uppercase and validate format (ISO 4217)
	currencyCode := strings.ToUpper(strings.TrimSpace(currency))

	// Basic currency code validation (3 uppercase letters)
	if len(currencyCode) != 3 || !isAlpha(currencyCode) {
		err := fmt.Errorf(
			"%s: invalid currency code format: %s (must be 3 uppercase letters)",
			op,
			currencyCode,
		)
		s.logger.Error(err.Error())
		return nil, err
	}

	// Create money amount from the smallest unit (e.g., cents for USD)
	moneyAmount, err := money.NewFromSmallestUnit(amount, money.Code(currencyCode))
	if err != nil {
		err = fmt.Errorf(
			"%s: failed to create money amount from %d %s: %w",
			op,
			amount,
			currencyCode,
			err,
		)
		s.logger.Error(err.Error(),
			"amount", amount,
			"currency", currencyCode,
		)
		return nil, err
	}

	return moneyAmount, nil
}

// parseProviderFeeAmount parses the provider fee amount with validation.
func (s *StripePaymentProvider) parseProviderFeeAmount(
	feeAmount int64,
	cur string,
	log *slog.Logger,
) (*money.Money, error) {
	// Validate currency code
	if cur == "" {
		err := fmt.Errorf("empty currency code provided for fee amount %d", feeAmount)
		log.Error("invalid currency code", "error", err)
		return nil, fmt.Errorf("invalid currency code: %w", err)
	}

	// Convert to uppercase to ensure consistency
	currency := strings.ToUpper(cur)

	// Log the fee being processed for debugging
	log = log.With(
		"fee_amount", feeAmount,
		"fee_currency", currency,
	)

	// Create money object with validated currency
	fee, err := s.parseAmount(feeAmount, currency)
	if err != nil {
		err = fmt.Errorf("invalid fee amount %d %s: %w", feeAmount, currency, err)
		log.Error("error parsing fee amount", "error", err)
		return nil, fmt.Errorf("error parsing fee amount: %w", err)
	}

	log.Debug("successfully parsed provider fee")
	return fee, nil
}

// buildPaymentCompletedEventPayload creates a PaymentCompleted event
// with the given amount and metadata.
// It ensures the event is properly constructed without triggering PaymentInitiated handlers.
func (s *StripePaymentProvider) buildPaymentCompletedEventPayload(
	amount *money.Money,
	paymentID string,
	meta *metadataInfo,
	log *slog.Logger,
) *events.PaymentCompleted {
	// Create a new PaymentCompleted event with minimal required fields
	pc := events.NewPaymentCompleted(
		&events.FlowEvent{
			ID:            uuid.New(),
			FlowType:      "payment",
			UserID:        meta.UserID,
			AccountID:     meta.AccountID,
			CorrelationID: meta.TransactionID,
			Timestamp:     time.Now(),
		},
		func(pc *events.PaymentCompleted) {
			pc.TransactionID = meta.TransactionID
			pc.PaymentID = &paymentID
			pc.Amount = amount
			pc.Status = "completed"
		},
	)

	// Set payment ID if available
	if meta.PaymentID != "" {
		pc.PaymentID = &meta.PaymentID
	}

	log.Info("built payment completed event",
		"event_id", pc.ID,
		"transaction_id", meta.TransactionID,
		"amount", amount.Amount(),
		"currency", amount.Currency(),
	)

	return pc
}

func (s *StripePaymentProvider) handlePaymentIntentFailed(
	ctx context.Context,
	event stripe.Event, log *slog.Logger) (*payment.PaymentEvent, error) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		log.Error(
			"error parsing payment_intent.payment_failed",
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
			"error retrieving payment intent",
			"error", err,
		)
		return nil, fmt.Errorf("error retrieving payment intent: %w", err)
	}

	// Get the user ID, account ID, and transaction ID from metadata
	userID, err := uuid.Parse(paymentIntent.Metadata["user_id"])
	if err != nil {
		log.Error(
			"invalid user_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	accountID, err := uuid.Parse(paymentIntent.Metadata["account_id"])
	if err != nil {
		log.Error(
			"invalid account_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid account_id in metadata: %w", err)
	}

	transactionID, err := uuid.Parse(
		paymentIntent.Metadata["transaction_id"])
	if err != nil {
		log.Error(
			"invalid transaction_id in metadata",
			"error", err,
			"metadata", paymentIntent.Metadata,
		)
		return nil, fmt.Errorf("invalid transaction_id in metadata: %w", err)
	}

	// Create metadata map from payment intent metadata
	metadata := make(map[string]string)
	maps.Copy(metadata, paymentIntent.Metadata)

	if err := s.bus.Emit(ctx, events.NewPaymentFailed(
		&events.FlowEvent{
			ID:            transactionID,
			UserID:        userID,
			AccountID:     accountID,
			FlowType:      "payment",
			CorrelationID: uuid.New(),
		},
		events.WithFailedPaymentID(&pi.ID),
	)); err != nil {
		log.Error(
			"error emitting payment failed event",
			"error", err,
		)
		return nil, fmt.Errorf("error emitting payment failed event: %w", err)
	}

	log.Info(
		"✅ Payment intent failed and transaction updated",
		"transaction_id", transactionID,
		"payment_id", paymentIntent.ID,
	)

	return &payment.PaymentEvent{
		ID:        pi.ID,
		Status:    payment.PaymentFailed,
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

func (s *StripePaymentProvider) handleChargeSucceeded(
	ctx context.Context,
	event stripe.Event,
	logger *slog.Logger,
) (
	*payment.PaymentEvent,
	error,
) {
	var charge stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		logger.Error(
			"error parsing charge.succeeded",
			"error", err,
		)
		return nil, fmt.Errorf(
			"error parsing charge.succeeded: %w", err)
	}
	// Always attempt to retrieve the Stripe fee from the balance transaction.
	balanceTxID := ""
	if charge.BalanceTransaction != nil {
		balanceTxID = charge.BalanceTransaction.ID
	}
	feeAmount := int64(0)
	feeCurrency := string(charge.Currency)
	var feeErr error
	if balanceTxID != "" {
		feeAmount, feeCurrency, feeErr = s.getFeeFromBalanceTransaction(ctx, logger, balanceTxID)
		if feeErr != nil {
			logger.Warn("Failed to retrieve fee from balance transaction", "error", feeErr)
			feeAmount = 0
			feeCurrency = string(charge.Currency)
		}
	} else {
		logger.Warn("No balance transaction found on Charge, defaulting fee to 0")
	}
	if feeCurrency == "" {
		feeCurrency = string(charge.Currency)
	}
	feeCurrency = strings.ToUpper(feeCurrency)
	logger = logger.With("charge_id", charge.ID, "balance_transaction_id", balanceTxID)
	logger.Info("✅ Charge succeeded", "fee_amount", feeAmount, "fee_currency", feeCurrency)
	// Process and emit fee if metadata is valid
	if feeEvent, err := s.createFeeEvent(
		charge.Metadata,
		feeAmount,
		feeCurrency,
		logger,
	); err == nil {
		logger.Info("Emitting FeesCalculated event", "event", feeEvent)
		_ = s.bus.Emit(ctx, feeEvent)
	}
	return nil, nil
}

// createFeeEvent creates a FeesCalculated event from the given
// transaction metadata and fee details.
// It returns the created event or an error if any required metadata is missing or invalid.
func (s *StripePaymentProvider) createFeeEvent(
	metadata map[string]string,
	feeAmount int64,
	feeCurrency string,
	logger *slog.Logger,
) (*events.FeesCalculated, error) {
	// Validate metadata exists
	if len(metadata) == 0 {
		return nil, fmt.Errorf("missing required metadata")
	}

	// Parse required metadata fields
	userID, err := uuid.Parse(metadata["user_id"])
	if err != nil {
		logger.Error("Failed to parse user_id from metadata", "error", err)
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	accountID, err := uuid.Parse(metadata["account_id"])
	if err != nil {
		logger.Error("Failed to parse account_id from metadata", "error", err)
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	transactionID, err := uuid.Parse(metadata["transaction_id"])
	if err != nil {
		logger.Error("Failed to parse transaction_id from metadata", "error", err)
		return nil, fmt.Errorf("invalid transaction_id: %w", err)
	}

	// Parse fee amount into money type
	feeMoney, err := s.parseProviderFeeAmount(feeAmount, feeCurrency, logger)
	if err != nil {
		logger.Error("Failed to parse provider fee amount",
			"amount", feeAmount,
			"currency", feeCurrency,
			"error", err)
		return nil, fmt.Errorf("invalid fee amount: %w", err)
	}

	logger.Debug("Creating fee event",
		"user_id", userID,
		"account_id", accountID,
		"transaction_id", transactionID,
		"fee_amount", feeMoney.Amount(),
		"fee_currency", feeMoney.Currency().String())

	// Create and return the fee event
	feeEvent := events.NewFeesCalculated(
		&events.FlowEvent{
			ID:            uuid.New(),
			UserID:        userID,
			AccountID:     accountID,
			FlowType:      "payment",
			CorrelationID: transactionID,
			Timestamp:     time.Now(),
		},
		events.WithFeeAmountValue(feeMoney),
		events.WithFeeTransactionID(transactionID),
		events.WithFeeType(account.FeeProvider),
	)

	return feeEvent, nil
}

// InitiatePayout implements payment.Payment interface
func (s *StripePaymentProvider) InitiatePayout(
	ctx context.Context,
	params *payment.InitiatePayoutParams,
) (*payment.InitiatePayoutResponse, error) {
	// First try to get the account with capabilities expanded

	s.logger.Info("Initiating payout",
		"user_id", params.UserID,
		"amount", params.Amount,
		"currency", params.Currency,
		"destination_type", params.Destination.Type,
		"destination_id", params.PaymentProviderID,
	)

	// Create the transfer to the connected account
	transferParams := &stripe.TransferCreateParams{
		Amount:      stripe.Int64(params.Amount),
		Currency:    stripe.String(params.Currency),
		Destination: stripe.String(params.PaymentProviderID),
		Description: stripe.String(params.Description),
	}

	// Add metadata
	transferParams.AddMetadata("user_id", params.UserID.String())
	transferParams.AddMetadata("account_id", params.AccountID.String())
	transferParams.AddMetadata("transaction_id", params.TransactionID.String())

	// Add any additional metadata
	for k, v := range params.Metadata {
		transferParams.AddMetadata(k, v)
	}

	// Execute the transfer
	transfer, err := s.client.V1Transfers.Create(ctx, transferParams)
	if err != nil {
		s.logger.Error("failed to create transfer",
			"error", err,
			"user_id", params.UserID,
			"account_id", params.AccountID,
			"stripe_account_id", params.PaymentProviderID)
		return nil, fmt.Errorf("failed to create transfer: %w", err)
	}

	// Determine the status based on the transfer status
	status := payment.PaymentPending
	if transfer.Reversed {
		status = payment.PaymentFailed
	} else if transfer.DestinationPayment != nil && len(transfer.DestinationPayment.ID) > 0 {
		// If we have a destination payment, the transfer was successful
		status = payment.PaymentCompleted
	}

	// Get the fee amount if available
	feeAmount := int64(0)
	if transfer.DestinationPayment != nil {
		feeAmount = max(transfer.DestinationPayment.Amount-transfer.Amount, 0)
	}

	return &payment.InitiatePayoutResponse{
		PayoutID:             transfer.ID,
		PaymentProviderID:    params.PaymentProviderID,
		Status:               status,
		Amount:               transfer.Amount,
		Currency:             string(transfer.Currency),
		FeeAmount:            feeAmount,
		FeeCurrency:          string(transfer.Currency),
		EstimatedArrivalDate: transfer.Created + 2*24*60*60, // Default to 2 days from creation
	}, nil
}
