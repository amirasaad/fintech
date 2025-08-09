package payment

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleInitiated(t *testing.T) {
	var err error

	t.Run("returns error for unexpected event type", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		h.WithHandler(
			HandleInitiated(
				h.Bus,
				h.MockPaymentProvider,
				h.Logger,
			),
		)
		t.Logf("Handler: %+v, Ctx: %+v, Event: %+v", h.Handler, h.Ctx, &testutils.TestEvent{})
		err = h.Handler(h.Ctx, &testutils.TestEvent{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
	})

	t.Run("handles payment initiation failure", func(t *testing.T) {
		t.Parallel()
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

		h.WithHandler(HandleInitiated(
			h.Bus,
			h.MockPaymentProvider,
			h.Logger,
		))
		h.MockPaymentProvider.EXPECT().
			InitiatePayment(
				h.Ctx,
				&provider.InitiatePaymentParams{
					UserID:        event.UserID,
					AccountID:     event.AccountID,
					Amount:        event.Amount.Amount(),
					Currency:      event.Amount.Currency().String(),
					TransactionID: event.TransactionID,
				},
			).Return(nil, errors.New("payment provider error"))
		err = h.Handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payment provider error")
	})

	t.Run("successfully initiates payment", func(t *testing.T) {
		t.Parallel()
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

		h.WithHandler(HandleInitiated(
			h.Bus,
			h.MockPaymentProvider,
			h.Logger,
		))
		h.MockPaymentProvider.EXPECT().
			InitiatePayment(
				h.Ctx,
				&provider.InitiatePaymentParams{
					UserID:        event.UserID,
					AccountID:     event.AccountID,
					Amount:        event.Amount.Amount(),
					Currency:      event.Amount.Currency().String(),
					TransactionID: event.TransactionID,
				},
			).Return(&provider.InitiatePaymentResponse{},
			nil,
		)
		err = h.Handler(h.Ctx, event)
		require.NoError(t, err)
	})

	t.Run("skips already processed payment initiated event", func(t *testing.T) {
		t.Parallel()
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

		// Simulate event already processed
		processedPaymentInitiated.Store(event.TransactionID.String(), struct{}{})

		h.WithHandler(HandleInitiated(h.Bus, h.MockPaymentProvider, h.Logger))
		err = h.Handler(h.Ctx, event)
		require.NoError(t, err)

		// Clean up for subsequent tests
		processedPaymentInitiated.Delete(event.TransactionID.String())
	})
}
