package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/testutils"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

	t.Run("handles nil payment ID", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.Bus, h.UOW, h.Logger)

		event := events.NewPaymentCompleted(
			&events.FlowEvent{
				ID:            h.EventID,
				CorrelationID: h.CorrelationID,
				FlowType:      "payment",
			},
			func(pc *events.PaymentCompleted) {
				pc.PaymentID = nil // Explicitly nil
				pc.TransactionID = h.TransactionID
				pc.Amount = h.Amount
			},
		)

		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repoaccount.Repository)(nil)).
					Return(h.MockAccRepo, nil).
					Once()
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				return fn(h.UOW)
			}).
			Return(errors.New("payment ID is nil")).
			Once()

		err := handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "payment ID is nil")
	})

	t.Run("gracefully skips when transaction not found", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.Bus, h.UOW, h.Logger)

		event := createValidPaymentCompletedEvent(h)

		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
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

	t.Run("updates payment ID when transaction exists without payment ID", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.Bus, h.UOW, h.Logger)

		event := createValidPaymentCompletedEvent(h)
		paymentID := "test-payment-id"

		tx := &dto.TransactionRead{
			ID:        h.TransactionID,
			UserID:    h.UserID,
			AccountID: h.AccountID,
			PaymentID: nil, // No payment ID initially
			Status:    string(account.TransactionStatusPending),
			Currency:  "USD",
			Amount:    h.Amount.AmountFloat(),
		}

		testAccount := &dto.AccountRead{
			ID:       h.AccountID,
			UserID:   h.UserID,
			Balance:  h.Amount.AmountFloat(),
			Currency: "USD",
		}

		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repoaccount.Repository)(nil)).
					Return(h.MockAccRepo, nil).
					Once()
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, paymentID).
					Return(tx, nil).
					Once()

				// Expect payment ID update
				h.MockTxRepo.EXPECT().
					Update(
						ctx,
						h.TransactionID,
						mock.MatchedBy(func(update dto.TransactionUpdate) bool {
							return update.PaymentID != nil && *update.PaymentID == paymentID
						})).
					Return(nil).
					Once()

				h.MockAccRepo.EXPECT().
					Get(h.Ctx, h.AccountID).
					Return(testAccount, nil).
					Once()

				h.MockTxRepo.EXPECT().
					Update(
						ctx,
						h.TransactionID,
						mock.MatchedBy(func(update dto.TransactionUpdate) bool {
							return update.Status != nil && *update.Status == "completed"
						})).
					Return(nil).
					Once()

				h.MockAccRepo.EXPECT().
					Update(h.Ctx, h.AccountID, mock.Anything).
					Return(nil).
					Once()

				return fn(h.UOW)
			}).
			Return(nil).
			Once()

		err := handler(h.Ctx, event)
		require.NoError(t, err)
	})

	t.Run("handles account mapping error", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.Bus, h.UOW, h.Logger)

		event := createValidPaymentCompletedEvent(h)
		paymentID := "test-payment-id"

		tx := &dto.TransactionRead{
			ID:        h.TransactionID,
			UserID:    h.UserID,
			AccountID: h.AccountID,
			PaymentID: &paymentID,
			Status:    string(account.TransactionStatusPending),
			Currency:  "INVALID", // Invalid currency to cause mapping error
			Amount:    h.Amount.AmountFloat(),
		}

		testAccount := &dto.AccountRead{
			ID:       h.AccountID,
			UserID:   h.UserID,
			Balance:  h.Amount.AmountFloat(),
			Currency: "INVALID", // Invalid currency
		}

		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repoaccount.Repository)(nil)).
					Return(h.MockAccRepo, nil).
					Once()
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, paymentID).
					Return(tx, nil).
					Once()

				h.MockAccRepo.EXPECT().
					Get(h.Ctx, h.AccountID).
					Return(testAccount, nil).
					Once()

				return fn(h.UOW)
			}).
			Return(errors.New("failed to map account to domain")).
			Once()

		err := handler(h.Ctx, event)
		require.Error(t, err)
	})

	t.Run("handles account balance update error", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.Bus, h.UOW, h.Logger)

		event := createValidPaymentCompletedEvent(h)
		paymentID := "test-payment-id"

		tx := &dto.TransactionRead{
			ID:        h.TransactionID,
			UserID:    h.UserID,
			AccountID: h.AccountID,
			PaymentID: &paymentID,
			Status:    string(account.TransactionStatusPending),
			Currency:  "USD",
			Amount:    h.Amount.AmountFloat(),
		}

		testAccount := &dto.AccountRead{
			ID:       h.AccountID,
			UserID:   h.UserID,
			Balance:  h.Amount.AmountFloat(),
			Currency: "USD",
		}

		updateErr := errors.New("failed to update account balance")

		h.UOW.EXPECT().
			Do(h.Ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) error {
				h.UOW.EXPECT().
					GetRepository((*repoaccount.Repository)(nil)).
					Return(h.MockAccRepo, nil).
					Once()
				h.UOW.EXPECT().
					GetRepository((*repotransaction.Repository)(nil)).
					Return(h.MockTxRepo, nil).
					Once()

				h.MockTxRepo.EXPECT().
					GetByPaymentID(h.Ctx, paymentID).
					Return(tx, nil).
					Once()

				h.MockAccRepo.EXPECT().
					Get(h.Ctx, h.AccountID).
					Return(testAccount, nil).
					Once()

				h.MockTxRepo.EXPECT().
					Update(h.Ctx, h.TransactionID, mock.Anything).
					Return(nil).
					Once()

				h.MockAccRepo.EXPECT().
					Update(h.Ctx, h.AccountID, mock.Anything).
					Return(updateErr).
					Once()

				return fn(h.UOW)
			}).
			Return(updateErr).
			Once()

		err := handler(h.Ctx, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update account balance")
	})
}
