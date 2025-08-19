package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
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

		// Create a test transaction
		testTx := &dto.TransactionRead{
			ID:        h.TransactionID,
			AccountID: h.AccountID,
			UserID:    h.UserID,
			Status:    "pending",
			Amount:    100.0,
			Currency:  "USD",
			PaymentID: h.PaymentID,
		}

		// Setup mocks for successful processing
		// Setup mocks for successful processing
		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				// Setup repository expectations for the transaction
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				// Expect GetByPaymentID to be called with the payment ID
				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, h.PaymentID).
					Return(testTx, nil).
					Once()

				// Expect Update to be called with the transaction ID and proper status
				h.MockTxRepo.EXPECT().
					Update(h.Ctx, h.TransactionID, mock.MatchedBy(
						func(update dto.TransactionUpdate) bool {
							return update.Status != nil && *update.Status == "processed"
						})).
					Return(nil).
					Once()

				// Execute the handler's callback
				return fn(h.UOW)
			}).
			Return(nil).
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
				paymentID := "test-payment-id"
				pp.PaymentID = &paymentID
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

		// Create a test transaction
		testTx := &dto.TransactionRead{
			ID:        h.TransactionID,
			AccountID: h.AccountID,
			UserID:    h.UserID,
			Status:    "pending",
			Amount:    100.0,
			Currency:  "USD",
			PaymentID: h.PaymentID,
		}

		// Create a valid payment processed event with transaction ID
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				paymentID := "test-payment-id"
				pp.PaymentID = &paymentID
			},
		)

		// Setup mocks to return an error from the transaction repository
		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				// Setup repository expectations for the transaction
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				// Expect GetByPaymentID to be called with the payment ID
				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, h.PaymentID).
					Return(testTx, nil).
					Once()

				// Expect Update to be called with the transaction ID and proper status
				h.MockTxRepo.EXPECT().
					Update(h.Ctx, h.TransactionID, mock.Anything).
					Return(errors.New("repository error")).
					Once()

				// Execute the handler's callback
				return fn(h.UOW)
			}).
			Return(errors.New("repository error")).
			Once()

		// Execute and verify error
		err := handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository error")
	})

	t.Run("handles successful transaction update", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		// Create a test transaction
		testTx := &dto.TransactionRead{
			ID:        h.TransactionID,
			AccountID: h.AccountID,
			UserID:    h.UserID,
			Status:    "pending",
			Amount:    100.0,
			Currency:  "USD",
			PaymentID: h.PaymentID,
		}

		// Create a valid payment processed event with transaction ID
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				paymentID := "test-payment-id"
				pp.PaymentID = &paymentID
			},
		)

		// Setup mocks for successful processing
		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				// Setup repository expectations for the transaction
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				// Expect GetByPaymentID to be called with the payment ID
				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, h.PaymentID).
					Return(testTx, nil).
					Once()

				// Expect Update to be called with the transaction ID and proper status
				h.MockTxRepo.EXPECT().
					Update(h.Ctx, h.TransactionID, mock.MatchedBy(
						func(update dto.TransactionUpdate) bool {
							return update.Status != nil && *update.Status == "processed"
						})).
					Return(nil).
					Once()

				// Execute the handler's callback
				return fn(h.UOW)
			}).
			Return(nil).
			Once()

		// Execute
		err := handler(h.Ctx, event)
		require.NoError(t, err)

		// Verify all expectations were met
		h.UOW.AssertExpectations(t)
		h.MockTxRepo.AssertExpectations(t)
	})
}
