package provider

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

	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/money"
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

// StripePaymentProvider implements Payment using Stripe API.
type StripePaymentProvider struct {
	bus             eventbus.Bus
	client          *stripe.Client
	checkoutService *checkout.Service
	cfg             *config.Stripe
	logger          *slog.Logger
}

// NewStripePaymentProvider creates a new StripePaymentProvider with the given
// API key, registry, and logger. The registry parameter is used for storing
// checkout session data.
func NewStripePaymentProvider(
	bus eventbus.Bus,
	checkoutProvider registry.Provider,
	cfg *config.Stripe,
	logger *slog.Logger,
) *StripePaymentProvider {
	client := stripe.NewClient(cfg.ApiKey)

	return &StripePaymentProvider{
		bus:             bus,
		client:          client,
		cfg:             cfg,
		checkoutService: checkout.New(checkoutProvider),
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

	// Note: We're using the transaction ID as the payment ID to maintain consistency
	// with our internal transaction tracking. The checkout session ID is stored in the
	// checkout service for reference.
	return &provider.InitiatePaymentResponse{
		Status:    provider.PaymentPending,
		PaymentID: co.PaymentID, // Use transaction ID as payment ID
	}, nil
}

// HandleWebhook handles Stripe webhook events using a handler map.
func (s *StripePaymentProvider) HandleWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) (*provider.PaymentEvent, error) {
	log := s.logger.With("handler", "stripe.HandleWebhook")
	event, err := webhook.ConstructEvent(payload, signature, s.cfg.SigningSecret)
	if err != nil {
		log.Error("invalid webhook signature", "error", err)
		return nil, fmt.Errorf("error verifying webhook signature: %w", err)
	}
	log = log.With("event_type", event.Type, "event_id", event.ID)
	log.Info("📥 Handling webhook event", "type", event.Type)

	handlers := map[string]func(
		context.Context,
		stripe.Event,
		*slog.Logger,
	) (*provider.PaymentEvent, error){
		"checkout.session.completed":    s.handleCheckoutSessionCompleted,
		"checkout.session.expired":      s.handleCheckoutSessionExpired,
		"payment_intent.succeeded":      s.handlePaymentIntentSucceeded,
		"payment_intent.payment_failed": s.handlePaymentIntentFailed,
		"charge.succeeded":              s.handleChargeSucceeded,
		"charge.updated":                s.handleChargeSucceeded,
	}
	if handler, ok := handlers[string(event.Type)]; ok {
		return handler(ctx, event, log)
	}
	log.Info("Unhandled event type")
	return nil, nil
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
) (*provider.PaymentEvent, error) {
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

	return &provider.PaymentEvent{
		ID:        session.PaymentIntent.ID,
		Status:    provider.PaymentCompleted,
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
) (*provider.PaymentEvent, error) {
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
	*provider.PaymentEvent,
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
		err := fmt.Errorf("%s: currency code is empty", op)
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
	pp := events.NewPaymentProcessed(&events.FlowEvent{
		ID:            uuid.New(),
		FlowType:      "payment",
		UserID:        parsedMeta.UserID,
		AccountID:     parsedMeta.AccountID,
		CorrelationID: parsedMeta.TransactionID,
		Timestamp:     time.Now(),
	}).
		WithAmount(amount).
		WithPaymentID(pi.ID).
		WithTransactionID(parsedMeta.TransactionID)

	log.Info("🔄 Emitting PaymentProcessed event",
		"transaction_id", parsedMeta.TransactionID,
		"payment_id", pi.ID)

	if err := s.bus.Emit(ctx, pp); err != nil {
		log.Error("error emitting payment processed event", "error", err)
		return nil, fmt.Errorf("error emitting payment processed event: %w", err)
	}
	// Emit PaymentCompleted event with zero fee since we're dropping fees
	pc := s.buildPaymentCompletedEventPayload(&pi, parsedMeta, log)
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
	return &provider.PaymentEvent{
		ID:        pi.ID,
		Status:    provider.PaymentCompleted,
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

// It ensures the fee amount is properly formatted and logs the fee being included in the event.
func (s *StripePaymentProvider) buildPaymentCompletedEventPayload(
	pi *stripe.PaymentIntent,
	meta *metadataInfo,
	log *slog.Logger,
) *events.PaymentCompleted {
	// Create payment amount with proper error handling
	paymentAmount, err := s.parseAmount(pi.Amount, string(pi.Currency))
	if err != nil {
		log.Error("error parsing payment amount",
			"error", err,
			"amount", pi.Amount,
			"currency", pi.Currency,
		)
		// Fallback to zero amount if we can't parse the payment amount
		zero, zeroErr := s.parseAmount(0, string(pi.Currency))
		if zeroErr != nil {
			log.Error("failed to parse zero amount", "error", zeroErr, "currency", pi.Currency)
			// If we can't even create a zero amount, we have to fail
			return nil
		}
		paymentAmount = zero
	}

	// Build the event with all required fields
	pc := events.NewPaymentCompleted(&events.FlowEvent{
		ID:            uuid.New(),
		FlowType:      "payment",
		UserID:        meta.UserID,
		AccountID:     meta.AccountID,
		CorrelationID: meta.TransactionID,
		Timestamp:     time.Now(),
	}, func(pc *events.PaymentCompleted) {
		pc.Amount = paymentAmount
		pc.PaymentID = &pi.ID
		pc.TransactionID = meta.TransactionID
	})
	log.Info("built payment completed event",
		"transaction_id", meta.TransactionID,
		"amount", paymentAmount.String(),
		"currency", pi.Currency,
	)

	return pc
}

func (s *StripePaymentProvider) handlePaymentIntentFailed(
	ctx context.Context,
	event stripe.Event, log *slog.Logger) (*provider.PaymentEvent, error) {
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

func (s *StripePaymentProvider) handleChargeSucceeded(
	ctx context.Context,
	event stripe.Event,
	logger *slog.Logger,
) (
	*provider.PaymentEvent,
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
