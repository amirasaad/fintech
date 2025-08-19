package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompletedHandler(t *testing.T) {
	// Helper function to create a properly initialized test helper
	newTestHelper := func(t *testing.T) *testutils.TestHelper {
		// First create the helper with minimal options
		h := testutils.New(t)

		// Initialize required fields
		amount, err := money.New(10.00, money.USD)
		require.NoError(t, err)

		feeAmount, err := money.New(1.00, money.USD)
		require.NoError(t, err)

		// Set the amounts using the helper methods
		h = h.WithAmount(amount).WithFeeAmount(feeAmount)

		return h
	}

	t.Run("successfully processes payment completion", func(t *testing.T) {
		h := newTestHelper(t)
		event := createValidPaymentCompletedEvent(h)

		// Setup test data and mocks using the helper function
		setupSuccessfulTest(h)

		// Call the handler
		handlerErr := h.WithHandler(
			HandleCompleted(h.Bus, h.UOW, h.Logger),
		).Handler(h.Ctx, event)
		require.NoError(t, handlerErr)
		h.AssertExpectations()
	})

	t.Run("returns nil for incorrect event type", func(t *testing.T) {
		h := newTestHelper(t)
		err := h.WithHandler(
			HandleCompleted(h.Bus, h.UOW, h.Logger),
		).Handler(h.Ctx, &testutils.TestEvent{})
		require.NoError(t, err)
	})

	t.Run("handles error from unit of work", func(t *testing.T) {
		h := newTestHelper(t)
		event := createValidPaymentCompletedEvent(h)

		// Setup mock expectations
		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			Return(errors.New("unit of work error")).
			Once()

		handlerErr := h.WithHandler(
			HandleCompleted(h.Bus, h.UOW, h.Logger),
		).Handler(h.Ctx, event)
		require.Error(t, handlerErr)
		assert.Contains(t, handlerErr.Error(), "unit of work error")
	})

	t.Run("handles error getting transaction by payment ID", func(t *testing.T) {
		h := newTestHelper(t)
		event := createValidPaymentCompletedEvent(h)
		expectedErr := errors.New("record not found")

		doFn := func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
			h.UOW.EXPECT().
				GetRepository((*repoaccount.Repository)(nil)).
				Return(h.MockAccRepo, nil).
				Once()

			h.UOW.EXPECT().
				GetRepository((*repotransaction.Repository)(nil)).
				Return(h.MockTxRepo, nil).
				Once()

			paymentID := "test-payment-id"
			h.MockTxRepo.EXPECT().
				GetByPaymentID(h.Ctx, paymentID).
				Return(nil, expectedErr).
				Once()

			err := fn(h.UOW)
			require.ErrorIs(t, err, expectedErr)
			return err
		}

		h.UOW.EXPECT().Do(h.Ctx, mock.Anything).RunAndReturn(doFn).Once()

		handlerErr := h.WithHandler(
			HandleCompleted(h.Bus, h.UOW, h.Logger),
		).Handler(h.Ctx, event)

		require.Error(t, handlerErr)
		assert.ErrorIs(t, handlerErr, expectedErr)
	})

	t.Run("handles error getting account", func(t *testing.T) {
		t.Parallel()
		h := testutils.New(t)
		handler := HandleCompleted(h.Bus, h.UOW, h.Logger)
		expectedErr := errors.New("account not found")

		tx := &dto.TransactionRead{
			ID:        h.TransactionID,
			UserID:    h.UserID,
			AccountID: h.AccountID,
			PaymentID: h.PaymentID,
			Status:    string(account.TransactionStatusPending),
			Currency:  "USD",
			Amount:    h.Amount.AmountFloat(),
		}

		doFn := func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
			h.UOW.EXPECT().GetRepository(
				(*repotransaction.Repository)(nil)).Return(h.MockTxRepo, nil).Once()
			// Ensure we have a valid payment ID
			paymentID := "test-payment-id"
			h.PaymentID = &paymentID
			tx.PaymentID = &paymentID
			h.MockTxRepo.EXPECT().GetByPaymentID(h.Ctx, paymentID).Return(tx, nil).Once()

			h.UOW.EXPECT().GetRepository(
				(*repoaccount.Repository)(nil)).Return(h.MockAccRepo, nil).Once()
			h.MockAccRepo.EXPECT().Get(h.Ctx, h.AccountID).Return(nil, expectedErr).Once()

			err := fn(h.UOW)
			require.ErrorIs(t, err, expectedErr)
			return err
		}

		h.UOW.EXPECT().Do(h.Ctx, mock.Anything).RunAndReturn(doFn).Once()

		err := handler(h.Ctx, createValidPaymentCompletedEvent(h))
		require.ErrorIs(t, err, expectedErr)
	})
}
