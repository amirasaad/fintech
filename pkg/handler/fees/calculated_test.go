package fees

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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCalculated(t *testing.T) {
	ctx := context.Background()

	t.Run(
		"successfully applies fees to transaction and updates account balance",
		func(t *testing.T) {
			// Create test account
			acc := &dto.AccountRead{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Balance:  testutils.DefaultAmount,
				Currency: money.USD.String(),
			}
			paymentID := "pm_123456789"
			// Create test transaction
			tx := &dto.TransactionRead{
				ID:        uuid.New(),
				AccountID: acc.ID,
				UserID:    acc.UserID,
				Status:    "completed",
				Amount:    testutils.DefaultAmount,
				Fee:       0, // Initial fee is 0
				Currency:  money.USD.String(),
				PaymentID: &paymentID,
			}
			fee := account.Fee{
				Amount: money.Must(
					testutils.DefaultFeeAmount,
					money.Code(tx.Currency).ToCurrency(),
				),
				Type: account.FeeProvider,
			}

			h := testutils.New(t).
				WithUserID(acc.UserID).
				WithAccountID(acc.ID).
				WithTransactionID(tx.ID).
				WithPaymentID(tx.PaymentID).
				WithFeeAmount(fee.Amount)

			// The actual balance calculation is tested in the fee calculator tests
			// We'll just verify the handler calls the expected methods with the right parameters

			// Set up UoW mock expectations

			// Set up repository mocks outside the UoW transaction
			h.UOW.EXPECT().
				GetRepository((*repotransaction.Repository)(nil)).
				Return(h.MockTxRepo, nil).
				Once()

			h.UOW.EXPECT().
				GetRepository((*repoaccount.Repository)(nil)).
				Return(h.MockAccRepo, nil).
				Once()

			h.UOW.EXPECT().
				Do(ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
				Run(func(
					ctx context.Context,
					fn func(uow repository.UnitOfWork) error,
				) {

					h.MockTxRepo.EXPECT().
						Get(ctx, tx.ID).
						Return(tx, nil).
						Once()

					h.MockTxRepo.EXPECT().
						Update(ctx, tx.ID, mock.AnythingOfType("dto.TransactionUpdate")).
						Return(nil).
						Once()

					h.MockAccRepo.EXPECT().
						Get(ctx, acc.ID).
						Return(acc, nil).
						Once()

					h.MockAccRepo.EXPECT().
						Update(
							ctx,
							acc.ID,
							mock.AnythingOfType("dto.AccountUpdate"),
						).
						Return(nil).
						Once()

					// Execute the transaction function
					err := fn(h.UOW)
					require.NoError(t, err)
				}).
				Return(nil).
				Once()

			event := events.NewFeesCalculated(
				&events.FlowEvent{
					ID:            h.EventID,
					CorrelationID: h.CorrelationID,
					FlowType:      "payment",
				},
				events.WithFeeTransactionID(h.TransactionID),
				events.WithFee(fee),
			)

			err := h.WithHandler(
				HandleCalculated(h.UOW, h.Logger),
			).Handler(ctx, event)

			// Verify results
			require.NoError(t, err)
		})

	t.Run("handles error updating account balance", func(t *testing.T) {
		transactionID := uuid.New()
		accountID := uuid.New()
		userID := uuid.New()
		event := &events.FeesCalculated{
			FlowEvent: events.FlowEvent{
				ID:        uuid.New(),
				FlowType:  "deposit",
				UserID:    userID,
				AccountID: accountID,
			},
			TransactionID: transactionID,
			Fee: account.Fee{
				Amount: money.Must(testutils.DefaultFeeAmount, money.USD.ToCurrency()),
				Type:   account.FeeProvider,
			},
		}

		tx := &dto.TransactionRead{
			ID:        transactionID,
			AccountID: accountID,
			UserID:    userID,
			Status:    "pending",
			Currency:  "USD",
		}

		acc := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  testutils.DefaultAmount,
			Currency: "USD",
		}

		expectedErr := errors.New("failed to update account balance")
		h := testutils.New(t).
			WithContext(ctx).
			WithAccountID(accountID).
			WithTransactionID(transactionID).
			WithUserID(userID)

		// Set up repository mocks outside the UoW transaction
		h.UOW.EXPECT().
			GetRepository((*repotransaction.Repository)(nil)).
			Return(h.MockTxRepo, nil).
			Once()

		h.UOW.EXPECT().
			GetRepository((*repoaccount.Repository)(nil)).
			Return(h.MockAccRepo, nil).
			Once()

		h.UOW.EXPECT().
			Do(h.Ctx, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Run(func(ctx context.Context, fn func(uow repository.UnitOfWork) error) {
				// Set up repository mocks inside the UoW transaction
				h.MockTxRepo.EXPECT().
					Get(ctx, transactionID).
					Return(tx, nil).
					Once()

				// Expect transaction update
				h.MockTxRepo.EXPECT().
					Update(
						ctx,
						transactionID,
						mock.AnythingOfType("dto.TransactionUpdate"),
					).
					Return(nil).
					Once()

				h.MockAccRepo.EXPECT().
					Get(ctx, accountID).
					Return(acc, nil).
					Once()

				// Expect account update to fail
				h.MockAccRepo.EXPECT().
					Update(
						ctx,
						accountID,
						mock.AnythingOfType("dto.AccountUpdate"),
					).
					Return(expectedErr).
					Once()

				// Execute the transaction function
				err := fn(h.UOW)
				require.Error(t, err)
				// Check that the error contains the expected error message
				assert.Contains(t, err.Error(), expectedErr.Error())
			}).
			Return(expectedErr).
			Once()

		err := h.WithHandler(
			HandleCalculated(
				h.UOW,
				h.Logger)).Handler(ctx, event)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
