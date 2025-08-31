package withdraw

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/payment"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleValidated(t *testing.T) {
	// Setup test data
	userID := uuid.New()
	accountID := uuid.New()
	transactionID := uuid.New()
	correlationID := uuid.New()

	// Create a test logger
	logger := slog.Default()

	// Create test amounts
	amount, err := money.New(100.0, money.USD)
	require.NoError(t, err)

	// Create a test withdraw request
	req := &events.WithdrawRequested{
		FlowEvent: events.FlowEvent{
			ID:            uuid.New(),
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
		},
		Amount:            amount,
		BankAccountNumber: "123456789",
		RoutingNumber:     "987654321",
		TransactionID:     transactionID,
	}

	// Create a test withdraw validated event
	wv := &events.WithdrawValidated{
		WithdrawCurrencyConverted: events.WithdrawCurrencyConverted{
			CurrencyConverted: events.CurrencyConverted{
				CurrencyConversionRequested: events.CurrencyConversionRequested{
					FlowEvent: events.FlowEvent{
						ID:            uuid.New(),
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: correlationID,
						FlowType:      "withdraw",
					},
					OriginalRequest: req,
				},
				TransactionID:   transactionID,
				ConvertedAmount: amount,
			},
		},
	}

	t.Run("successful payout initiation", func(t *testing.T) {
		// Create mocks
		mockBus := mocks.NewBus(t)
		mockPayment := mocks.NewPaymentProvider(t)
		uow := mocks.NewUnitOfWork(t)

		// Mock user repository
		mockUserRepo := new(mocks.UserRepository)
		uow.On("GetRepository", mock.Anything).Return(mockUserRepo, nil)
		mockUserRepo.On("Get", mock.Anything, userID).Return(&dto.UserRead{
			ID:                     userID,
			Username:               "testuser",
			Email:                  "test@example.com",
			Names:                  "Test User",
			StripeConnectAccountID: "",
		}, nil)

		// Expected payout response
		expectedPayout := &payment.InitiatePayoutResponse{
			PayoutID:             "payout_123",
			PaymentProviderID:    "payout_123", // Ensure this matches what we expect in the test
			Status:               payment.PaymentPending,
			Amount:               amount.Amount(),
			Currency:             amount.Currency().String(),
			FeeAmount:            0,
			FeeCurrency:          amount.Currency().String(),
			EstimatedArrivalDate: time.Now().Add(24 * time.Hour).Unix(),
		}

		// Set up expectations
		mockPayment.On(
			"InitiatePayout",
			mock.Anything,
			mock.AnythingOfType("*payment.InitiatePayoutParams")).
			Return(expectedPayout, nil)

		// Expect user repository update with the payout ID
		mockUserRepo.On(
			"Update",
			mock.MatchedBy(func(ctx context.Context) bool { return true }), // Match any context
			userID,
			mock.MatchedBy(func(update *dto.UserUpdate) bool {
				return update.StripeConnectAccountID != nil &&
					*update.StripeConnectAccountID == expectedPayout.PaymentProviderID
			})).Return(nil)

		mockBus.EXPECT().
			Emit(
				mock.MatchedBy(func(ctx context.Context) bool { return true }), // Match any context
				mock.MatchedBy(func(e events.Event) bool {
					pp, ok := e.(*events.PaymentProcessed)
					if !ok {
						t.Logf("Unexpected event type: %T", e)
						return false
					}

					t.Logf("Event details: %+v", pp)
					t.Logf("TransactionID: got=%v, want=%v", pp.TransactionID, transactionID)
					t.Logf("UserID: got=%v, want=%v", pp.UserID, userID)
					t.Logf("AccountID: got=%v, want=%v", pp.AccountID, accountID)
					if pp.PaymentInitiated.PaymentID == nil {
						t.Log("PaymentID is nil")
					} else {
						t.Logf(
							"PaymentProviderID: got=%v, want=%v",
							*pp.PaymentInitiated.PaymentID, expectedPayout.PaymentProviderID)
					}

					return pp.TransactionID == transactionID &&
						pp.UserID == userID &&
						pp.AccountID == accountID &&
						pp.PaymentInitiated.PaymentID != nil &&
						*pp.PaymentInitiated.PaymentID == expectedPayout.PaymentProviderID
				}),
			).Return(nil)

		// Create handler
		handler := HandleValidated(mockBus, uow, mockPayment, logger)

		// Execute
		err := handler(context.Background(), wv)

		// Assert
		require.NoError(t, err)
	})
	t.Run("payout initiation failure", func(t *testing.T) {
		// Create mocks
		mockBus := mocks.NewBus(t)
		mockPayment := mocks.NewPaymentProvider(t)
		uow := mocks.NewUnitOfWork(t)

		// Mock user repository
		mockUserRepo := new(mocks.UserRepository)
		uow.On("GetRepository", mock.Anything).Return(mockUserRepo, nil)
		mockUserRepo.On("Get", mock.Anything, userID).Return(&dto.UserRead{
			ID:                     userID,
			Username:               "testuser",
			Email:                  "test@example.com",
			Names:                  "Test User",
			StripeConnectAccountID: "",
		}, nil)

		// Expected error
		expectedErr := errors.New("payout failed")

		// Set up expectations
		mockPayment.On("InitiatePayout", mock.Anything, mock.Anything).
			Return((*payment.InitiatePayoutResponse)(nil), expectedErr)

		// Expect a WithdrawFailed event
		mockBus.On("Emit", mock.Anything, mock.MatchedBy(func(e *events.WithdrawFailed) bool {
			return e.UserID == userID && e.AccountID == accountID
		})).Return(nil)

		// Create handler
		handler := HandleValidated(mockBus, uow, mockPayment, logger)

		// Execute
		err := handler(context.Background(), wv)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("invalid event type", func(t *testing.T) {
		// Create mocks
		mockBus := mocks.NewBus(t)
		uow := mocks.NewUnitOfWork(t)
		mockPayment := mocks.NewPaymentProvider(t)

		// Create handler
		handler := HandleValidated(mockBus, uow, mockPayment, logger)

		// Execute with wrong event type
		err := handler(context.Background(), &events.WithdrawRequested{})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected WithdrawValidated event")
	})
}
