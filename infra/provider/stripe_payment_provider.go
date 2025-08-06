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

	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/pkg/checkout"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
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

	if err := s.bus.Emit(
		ctx,
		events.NewPaymentProcessed(
			*events.NewPaymentInitiated(
				events.FlowEvent{
					ID:            uuid.New(),
					UserID:        se.UserID,
					AccountID:     se.AccountID,
					FlowType:      "payment",
					CorrelationID: uuid.New(),
				},
			), func(pp *events.PaymentProcessed) {
				pp.TransactionID = se.TransactionID
				pp.PaymentID = session.PaymentIntent.ID
				log.Info("Emitting ", "event_type", pp.Type())
			},
		),
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
		Amount:    session.PaymentIntent.Amount,
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
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		log.Error("parsing payment_intent.succeeded", "error", err)
		return nil, fmt.Errorf("error parsing payment_intent.succeeded: %w", err)
	}
	log = log.With("payment_intent_id", pi.ID)

	// Retrieve the expanded PaymentIntent with balance transaction
	fullPI, err := s.client.V1PaymentIntents.
		Retrieve(ctx, pi.ID, &stripe.PaymentIntentRetrieveParams{
			Params: stripe.Params{
				Expand: []*string{stripe.String("latest_charge.balance_transaction")},
			},
		})
	if err != nil {
		log.Error("retrieving expanded payment intent", "error", err)
		return nil, fmt.Errorf("error retrieving expanded payment intent: %w", err)
	}
	pi = *fullPI

	// Log Stripe fee and net info if available
	var feeAmount int64
	var feeCurrency string
	if pi.LatestCharge != nil && pi.LatestCharge.BalanceTransaction != nil {
		bt := pi.LatestCharge.BalanceTransaction
		log.Info("📊 Stripe balance transaction info",
			"fee", bt.Fee,
			"net", bt.Net,
			"currency", bt.Currency,
			"fee_details", bt.FeeDetails,
		)
		feeAmount = bt.Fee
		feeCurrency = strings.ToUpper(string(bt.Currency))
	} else {
		log.Warn("⚠️ Balance transaction not available via expand")
	}

	s.logStripeFeeInfo(log, &pi)

	parsedMeta, err := s.parseAndValidateMetadata(pi.Metadata, log)
	if err != nil {
		return nil, err
	}
	metadata := s.copyMetadata(pi.Metadata)

	log.Info("💰 Handling payment_intent.succeeded event", "payment_intent_id", pi.ID)

	fee, err := s.parseProviderFeeAmount(feeAmount, feeCurrency, log)
	if err != nil {
		return nil, err
	}

	pc := s.buildPaymentCompletedEventPayload(&pi, parsedMeta, fee, log)
	if err := s.bus.Emit(ctx, pc); err != nil {
		log.Error("error emitting payment completed event", "error", err)
		return nil, fmt.Errorf("error emitting payment completed event: %w", err)
	}

	log.Info("✅ Payment intent processed and transaction updated successfully",
		"transaction_id", parsedMeta.TransactionID, "payment_id", pi.ID)

	return &provider.PaymentEvent{
		ID:        pi.ID,
		Status:    provider.PaymentCompleted,
		Amount:    pi.Amount,
		Currency:  string(pi.Currency),
		UserID:    parsedMeta.UserID,
		AccountID: parsedMeta.AccountID,
		Metadata:  metadata,
	}, nil
}

// logStripeFeeInfo logs Stripe fee-related info.
func (s *StripePaymentProvider) logStripeFeeInfo(log *slog.Logger, pi *stripe.PaymentIntent) {
	log.Info("Stripe fees",
		"amount_received", pi.AmountReceived,
		"amount_details", pi.AmountDetails,
		"application_fee", pi.ApplicationFeeAmount,
		"currency", pi.Currency,
	)
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
	userID, err := uuid.Parse(meta["user_id"])
	if err != nil {
		log.Error(
			"invalid user_id in metadata",
			"error", err,
			"metadata", meta,
		)
		return nil, fmt.Errorf("invalid user_id in metadata: %w", err)
	}
	accountID, err := uuid.Parse(meta["account_id"])
	if err != nil {
		log.Error(
			"invalid account_id in metadata",
			"error", err,
			"metadata", meta,
		)
		return nil, fmt.Errorf("invalid account_id in metadata: %w", err)
	}
	transactionID, err := uuid.Parse(meta["transaction_id"])
	if err != nil {
		log.Error(
			"invalid transaction_id in metadata",
			"error", err,
			"metadata", meta,
		)
		return nil, fmt.Errorf(
			"invalid transaction_id in metadata: %w", err)
	}
	currencyCode := meta["currency"]
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
	copied := make(map[string]string)
	maps.Copy(copied, meta)
	return copied
}

// parseProviderFeeAmount parses the provider fee amount.
func (s *StripePaymentProvider) parseProviderFeeAmount(
	feeAmount int64,
	cur string,
	log *slog.Logger,
) (money.Money, error) {
	fee, err := money.NewFromSmallestUnit(
		feeAmount,
		currency.Code(cur),
	)
	if err != nil {
		log.Error("error creating money from smallest unit", "error", err)
		return money.Money{}, fmt.Errorf("error creating money from smallest unit: %w", err)
	}
	return fee, nil
}

// buildPaymentCompletedEventPayload builds the arguments for bus.Emit for PaymentCompleted.
func (s *StripePaymentProvider) buildPaymentCompletedEventPayload(
	pi *stripe.PaymentIntent,
	meta *metadataInfo,
	feeAmount money.Money,
	log *slog.Logger,
) *events.PaymentCompleted {
	amount, err := s.parseAmount(pi.AmountReceived, pi.Currency, log)
	return events.NewPaymentCompleted(
		events.FlowEvent{
			ID:            meta.TransactionID,
			UserID:        meta.UserID,
			AccountID:     meta.AccountID,
			FlowType:      "payment",
			CorrelationID: uuid.New(),
		},
		events.WithPaymentID(pi.ID),
		events.WithProviderFee(account.Fee{
			Amount: feeAmount,
			Type:   account.FeeProvider,
		}),
		func(pc *events.PaymentCompleted) {
			pc.TransactionID = meta.TransactionID
			pc.Amount = amount
			if err != nil {
				log.Error("error creating money from smallest unit", "error", err)
			}
		},
	)
}

// parseAmount parses the received amount and currency into a Money value.
func (s *StripePaymentProvider) parseAmount(
	amount int64,
	currencyCode stripe.Currency,
	log *slog.Logger,
) (money.Money, error) {
	return money.NewFromSmallestUnit(
		amount,
		currency.Code(strings.ToUpper(string(currencyCode))),
	)
}

// handlePaymentIntentFailed handles the payment_intent.payment_failed event
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
	bt, err := s.client.V1BalanceTransactions.Retrieve(ctx, charge.BalanceTransaction.ID, nil)
	if err != nil {
		return nil, err

	}
	logger = logger.With("charge_id", charge.ID, "balance_transaction_id", bt.ID)

	// Get the charge details
	logger.Info("✅ Charge succeeded", ", charge_id", charge.ID)
	return nil, nil
}
