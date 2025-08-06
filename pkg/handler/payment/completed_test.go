package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompletedHandler(t *testing.T) {
	t.Run("returns nil for incorrect event type", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.bus, h.uow, h.logger)

		// The handler should not call any repository methods for incorrect event types
		h.uow.EXPECT().GetRepository(mock.Anything).Unset()
		h.uow.EXPECT().Do(mock.Anything, mock.Anything).Unset()

		mockEvent := &testEvent{}
		err := handler(h.ctx, mockEvent)
		require.NoError(t, err)
	})

	t.Run("handles error from unit of work", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.bus, h.uow, h.logger)

		// Setup test transaction
		tx := &dto.TransactionRead{
			ID:        h.transactionID,
			UserID:    h.userID,
			AccountID: h.accountID,
			PaymentID: h.paymentID,
			Status:    string(account.TransactionStatusPending),
			Currency:  "USD",
			Amount:    h.amount.AmountFloat(),
		}

		// Setup test account
		testAccount := &dto.AccountRead{
			ID:       h.accountID,
			UserID:   h.userID,
			Balance:  h.amount.AmountFloat(), // $1000.00
			Currency: "USD",
		}

		// Mock the Do callback to return an error
		uowErr := errors.New("uow error")
		h.uow.EXPECT().Do(
			mock.Anything,
			mock.Anything,
		).Return(uowErr).Run(func(
			ctx context.Context,
			fn func(uow repository.UnitOfWork) error,
		) {
			// Setup UOW to return the mock repositories inside the callback
			h.uow.EXPECT().GetRepository(
				(*repoaccount.Repository)(nil),
			).Return(h.mockAccRepo, nil).Once()
			h.uow.EXPECT().GetRepository(
				(*repotransaction.Repository)(nil),
			).Return(h.mockTxRepo, nil).Once()

			// Mock the repositories
			h.mockTxRepo.EXPECT().GetByPaymentID(
				mock.Anything,
				h.paymentID,
			).Return(tx, nil).Once()
			h.mockAccRepo.EXPECT().Get(
				mock.Anything,
				h.accountID,
			).Return(testAccount, nil).Once()

			// Mock the Update methods since the handler will call them before the error occurs
			h.mockTxRepo.EXPECT().Update(
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(nil).Once()
			h.mockAccRepo.EXPECT().Update(
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(nil).Once()

			// The callback should not return an error
			err := fn(h.uow)
			require.NoError(t, err, "callback should not return error")
		}).Once()

		err := handler(h.ctx, h.createValidEvent())
		require.ErrorIs(t, err, uowErr)
	})

	t.Run("handles successful event", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		h.setupSuccessfulTest()
		handler := HandleCompleted(h.bus, h.uow, h.logger)

		err := handler(h.ctx, h.createValidEvent())
		require.NoError(t, err)
	})

	t.Run("handles error getting account repository", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.bus, h.uow, h.logger)
		expectedErr := errors.New("failed to get account repository")

		// Setup the Do method to execute the callback
		h.uow.EXPECT().Do(
			mock.Anything,
			mock.Anything,
		).Return(expectedErr).Run(func(
			ctx context.Context,
			fn func(uow repository.UnitOfWork) error,
		) {
			// The callback will call GetRepository for the account repository
			h.uow.EXPECT().GetRepository(
				(*repoaccount.Repository)(nil),
			).Return(nil, expectedErr).Once()

			// The callback should return the error from GetRepository
			err := fn(h.uow)
			require.ErrorIs(t, err, expectedErr)
		}).Once()

		err := handler(h.ctx, h.createValidEvent())
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("handles invalid account repository type", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.bus, h.uow, h.logger)

		// Setup the Do method to execute the callback
		h.uow.EXPECT().Do(
			mock.Anything,
			mock.Anything,
		).Return(ErrInvalidRepositoryType).Run(func(
			ctx context.Context,
			fn func(uow repository.UnitOfWork) error,
		) {
			// The callback will call GetRepository for the
			// account repository and return an invalid type
			h.uow.EXPECT().GetRepository(
				(*repoaccount.Repository)(nil),
			).Return("not a repository", nil).Once()

			// The callback should return the error from the type assertion
			err := fn(h.uow)
			require.ErrorIs(
				t,
				err,
				ErrInvalidRepositoryType,
			)
		}).Once()

		err := handler(h.ctx, h.createValidEvent())
		require.ErrorIs(t, err, ErrInvalidRepositoryType)
	})

	t.Run("handles error getting transaction by payment ID", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.bus, h.uow, h.logger)
		expectedErr := errors.New("database error")

		// Mock the Do callback and set up all repository interactions inside it
		h.uow.EXPECT().Do(
			mock.Anything,
			mock.Anything,
		).Return(expectedErr).Run(func(
			ctx context.Context,
			fn func(uow repository.UnitOfWork) error,
		) {
			// Setup UOW to return the mock repositories inside the callback
			h.uow.EXPECT().GetRepository(
				(*repoaccount.Repository)(nil),
			).Return(h.mockAccRepo, nil).Once()
			h.uow.EXPECT().GetRepository(
				(*repotransaction.Repository)(nil),
			).Return(h.mockTxRepo, nil).Once()

			// Mock the repositories to return an error when getting transaction by payment ID
			h.mockTxRepo.EXPECT().GetByPaymentID(
				mock.Anything,
				h.paymentID,
			).Return(nil, expectedErr).Once()

			// The callback should return the error from GetByPaymentID
			err := fn(h.uow)
			require.ErrorIs(
				t,
				err,
				expectedErr,
				"callback should return the expected error",
			)
		}).Once()

		err := handler(h.ctx, h.createValidEvent())
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("handles error getting account", func(t *testing.T) {
		t.Parallel()
		h := newTestHelper(t)
		handler := HandleCompleted(h.bus, h.uow, h.logger)
		expectedErr := errors.New("account not found")

		// Setup test transaction
		tx := &dto.TransactionRead{
			ID:        h.transactionID,
			UserID:    h.userID,
			AccountID: h.accountID,
			PaymentID: h.paymentID,
			Status:    string(account.TransactionStatusPending),
			Currency:  "USD",
			Amount:    h.amount.AmountFloat(),
		}

		// Mock the Do callback and set up all repository interactions inside it
		h.uow.EXPECT().Do(
			mock.Anything,
			mock.Anything,
		).Return(expectedErr).Run(
			func(
				ctx context.Context,
				fn func(uow repository.UnitOfWork) error,
			) {
				// The callback should be called with the uow

				// Setup UOW to return the mock repositories inside the callback
				h.uow.EXPECT().GetRepository(
					(*repoaccount.Repository)(nil),
				).Return(
					h.mockAccRepo,
					nil,
				).Once()
				h.uow.EXPECT().GetRepository(
					(*repotransaction.Repository)(nil),
				).Return(
					h.mockTxRepo,
					nil,
				).Once()

				// Setup mocks for repository methods
				h.mockTxRepo.EXPECT().GetByPaymentID(
					mock.Anything,
					h.paymentID,
				).Return(tx, nil).Once()
				h.mockAccRepo.EXPECT().Get(
					mock.Anything,
					h.accountID,
				).Return(nil, expectedErr).Once()

				// The callback should return the error from Get
				err := fn(h.uow)
				require.ErrorIs(
					t,
					err,
					expectedErr,
					"callback should return the expected error",
				)
			}).Once()

		err := handler(h.ctx, h.createValidEvent())
		require.ErrorIs(t, err, expectedErr)
	})
}
