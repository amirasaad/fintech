package payment

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	handlercommon "github.com/amirasaad/fintech/pkg/handler/common"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleInitiated(t *testing.T) {
	// Helper function to create a new test helper with required fields
	newTestHelper := func(t *testing.T) *testutils.TestHelper {
		h := testutils.New(t)

		// Initialize required fields if needed
		if h.Amount == nil {
			amount, err := money.New(10.00, "USD")
			require.NoError(t, err)
			h = h.WithAmount(amount)
		}

		return h
	}

	t.Run("returns error for unexpected event type", func(t *testing.T) {
		h := newTestHelper(t)
		h = h.WithHandler(
			HandleInitiated(
				h.Bus,
				h.MockPaymentProvider,
				h.Logger,
			),
		)
		t.Logf("Handler: %+v, Ctx: %+v, Event: %+v", h.Handler, h.Ctx, &testutils.TestEvent{})
		err := h.Handler(h.Ctx, &testutils.TestEvent{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
	})

	t.Run("handles payment initiation failure", func(t *testing.T) {
		h := newTestHelper(t)
		event := events.NewPaymentInitiated(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pi *events.PaymentInitiated) {
				pi.TransactionID = h.TransactionID
				pi.UserID = h.UserID
				pi.AccountID = h.AccountID
				pi.Amount = h.Amount
			},
		)

		h.WithHandler(HandleInitiated(
			h.Bus,
			h.MockPaymentProvider,
			h.Logger,
		))
		h.MockPaymentProvider.EXPECT().
			InitiatePayment(
				h.Ctx,
				&payment.InitiatePaymentParams{
					UserID:        event.UserID,
					AccountID:     event.AccountID,
					Amount:        event.Amount.Amount(),
					Currency:      event.Amount.Currency().String(),
					TransactionID: event.TransactionID,
				},
			).Return(nil, errors.New("payment provider error"))
		err := h.Handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payment provider error")
	})

	t.Run("successfully initiates payment", func(t *testing.T) {
		h := newTestHelper(t)
		event := events.NewPaymentInitiated(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pi *events.PaymentInitiated) {
				pi.TransactionID = h.TransactionID
				pi.UserID = h.UserID
				pi.AccountID = h.AccountID
				pi.Amount = h.Amount
			},
		)

		h.WithHandler(HandleInitiated(
			h.Bus,
			h.MockPaymentProvider,
			h.Logger,
		))
		// Create a mock response with the expected payment ID
		mockResponse := &payment.InitiatePaymentResponse{
			Status:    payment.PaymentPending,
			PaymentID: "pi_123456789", // Mock payment ID
		}

		h.MockPaymentProvider.EXPECT().
			InitiatePayment(
				h.Ctx,
				&payment.InitiatePaymentParams{
					UserID:        event.UserID,
					AccountID:     event.AccountID,
					Amount:        event.Amount.Amount(),
					Currency:      event.Amount.Currency().String(),
					TransactionID: event.TransactionID,
				},
			).Return(mockResponse, nil).Maybe()

		err := h.Handler(h.Ctx, event)
		require.NoError(t, err)
	})

	t.Run("skips already processed payment initiated event", func(t *testing.T) {
		h := testutils.New(t)
		event := events.NewPaymentInitiated(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pi *events.PaymentInitiated) {
				pi.TransactionID = h.TransactionID
				pi.UserID = h.UserID
				pi.AccountID = h.AccountID
				pi.Amount = h.Amount
			},
		)

		// Wrap handler with idempotency middleware to test the full flow
		handler := HandleInitiated(h.Bus, h.MockPaymentProvider, h.Logger)
		tracker := handlercommon.NewIdempotencyTracker()
		wrappedHandler := handlercommon.WithIdempotency(
			handler,
			tracker,
			ExtractPaymentInitiatedKey,
			"HandleInitiated",
			h.Logger,
		)

		mockResponse := &payment.InitiatePaymentResponse{
			Status:    payment.PaymentPending,
			PaymentID: "pi_123456789",
		}

		// Set up expectation for first call only
		h.MockPaymentProvider.EXPECT().
			InitiatePayment(
				h.Ctx,
				&payment.InitiatePaymentParams{
					UserID:        event.UserID,
					AccountID:     event.AccountID,
					Amount:        event.Amount.Amount(),
					Currency:      event.Amount.Currency().String(),
					TransactionID: event.TransactionID,
				},
			).Return(mockResponse, nil).Times(1)

		// First call - should execute handler
		err := wrappedHandler(h.Ctx, event)
		require.NoError(t, err)

		// Second call - should be skipped due to idempotency (no mock expectation)
		err = wrappedHandler(h.Ctx, event)
		require.NoError(t, err)

	})
}
