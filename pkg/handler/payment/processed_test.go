package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/repository"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleProcessed(t *testing.T) {
	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		// Create a test event with unexpected type
		event := &testutils.TestEvent{}

		// Execute
		err := handler(h.Ctx, event)

		// Assert
		require.Error(t, err)
		h.UOW.AssertNotCalled(
			t,
			"Do",
			h.Ctx,
			mock.Anything,
		)
		h.UOW.ExpectedCalls = nil
		h.MockTxRepo.ExpectedCalls = nil
	})

	t.Run("successfully processes payment", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		// Setup mocks for successful processing
		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			Return(nil).
			Run(func(
				ctx context.Context,
				fn func(uow repository.UnitOfWork) error,
			) {
				h.UOW.EXPECT().
					GetRepository(
						(*repotransaction.Repository)(nil),
					).
					Return(h.MockTxRepo, nil).
					Once()

				h.MockTxRepo.EXPECT().
					Update(h.Ctx, h.TransactionID, mock.Anything).
					Return(nil).
					Once()

				// Simulate the callback that would be called by the handler
				err := fn(h.UOW)
				require.NoError(
					t,
					err,
					"callback should not return error",
				)
			}).
			Once()

		// Create a valid payment processed event with transaction ID
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = h.PaymentID
			},
		)

		// Execute
		err := handler(h.Ctx, event)

		// Assert
		require.NoError(t, err)
		h.UOW.ExpectedCalls = nil
		h.MockTxRepo.ExpectedCalls = nil
	})

	t.Run("handles repository error", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		// Create a valid payment processed event with transaction ID first
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = h.PaymentID
			},
		)

		// Setup mocks to return an error from Update
		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			Return(errors.New("repository error")).
			Once()

		// Execute and verify error
		err := handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository error")
		h.UOW.ExpectedCalls = nil
		h.MockTxRepo.ExpectedCalls = nil
	})

	t.Run("handles successful transaction update", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)

		// Setup mocks for successful transaction update
		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			Return(nil).
			Run(func(
				ctx context.Context,
				fn func(uow repository.UnitOfWork) error) {
				h.UOW.EXPECT().
					GetRepository(
						(*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				h.MockTxRepo.EXPECT().
					Update(h.Ctx, h.TransactionID, mock.Anything).
					Return(nil).
					Once()

				// Simulate the callback that would be called by the handler
				err := fn(h.UOW)
				require.NoError(
					t, err, "callback should not return error")
			}).
			Once()

		// Create a valid payment processed event with transaction ID
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = h.PaymentID
			},
		)

		// Execute
		err := h.WithHandler(
			HandleProcessed(
				h.UOW,
				h.Logger),
		).Handler(h.Ctx, event)

		// Assert no error occurred
		assert.NoError(t, err)
		h.UOW.ExpectedCalls = nil
		h.MockTxRepo.ExpectedCalls = nil
	})
}
