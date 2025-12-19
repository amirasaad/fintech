package payment

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/repository"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

	t.Run("gracefully skips when transaction not found", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		paymentID := "test-payment-id"
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = &paymentID
			},
		)

		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				// Return not found for both payment ID and transaction ID
				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, paymentID).
					Return(nil, gorm.ErrRecordNotFound).
					Once()
				h.MockTxRepo.EXPECT().
					Get(h.Ctx, h.TransactionID).
					Return(nil, gorm.ErrRecordNotFound).
					Once()

				return fn(h.UOW)
			}).
			Return(nil).
			Once()

		err := handler(h.Ctx, event)
		require.NoError(t, err)
	})

	t.Run("creates new transaction when not found", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		paymentID := "test-payment-id"
		amount, err := money.New(100.0, money.USD)
		require.NoError(t, err)

		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = &paymentID
				pp.UserID = h.UserID
				pp.AccountID = h.AccountID
				pp.Amount = amount
			},
		)

		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				// Return not found
				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, paymentID).
					Return(nil, gorm.ErrRecordNotFound).
					Once()
				h.MockTxRepo.EXPECT().
					Get(h.Ctx, h.TransactionID).
					Return(nil, gorm.ErrRecordNotFound).
					Once()

				// Expect UpsertByPaymentID to be called
				h.MockTxRepo.EXPECT().
					UpsertByPaymentID(
						h.Ctx,
						paymentID,
						mock.MatchedBy(func(create dto.TransactionCreate) bool {
							return create.ID == h.TransactionID &&
								create.UserID == h.UserID &&
								create.AccountID == h.AccountID &&
								create.Status == "processed" &&
								create.MoneySource == "Stripe"
						}),
					).
					Return(nil).
					Once()

				return fn(h.UOW)
			}).
			Return(nil).
			Once()

		err = handler(h.Ctx, event)
		require.NoError(t, err)
	})

	t.Run("handles error getting transaction repository", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		paymentID := "test-payment-id"
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = &paymentID
			},
		)

		expectedErr := errors.New("repository error")
		wrappedErr := fmt.Errorf("failed to get transaction repo: %w", expectedErr)
		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(nil, expectedErr).
					Once()

				return fn(h.UOW)
			}).
			Return(wrappedErr).
			Once()

		err := handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get transaction repo")
	})

	t.Run("handles lookup error", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleProcessed(h.UOW, h.Logger)

		paymentID := "test-payment-id"
		event := events.NewPaymentProcessed(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pp *events.PaymentProcessed) {
				pp.TransactionID = h.TransactionID
				pp.PaymentID = &paymentID
			},
		)

		lookupErr := errors.New("lookup error")
		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, paymentID).
					Return(nil, lookupErr).
					Once()

				return fn(h.UOW)
			}).
			Return(lookupErr).
			Once()

		err := handler(h.Ctx, event)
		require.Error(t, err)
		assert.Equal(t, lookupErr, err)
	})
}
