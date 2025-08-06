package payment

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockEvent struct{}

func (m *mockEvent) Type() string { return "mockEvent" }

func TestCompletedHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	bus := mocks.NewBus(t)
	mUow := mocks.NewUnitOfWork(t)

	// Generate consistent test IDs
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testAccountID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testPaymentID := "pay_123"
	testEventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testCorrelationID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	testTransactionID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	// Create test money amount
	amount, err := money.New(10000, "USD") // 100.00 USD in cents
	if err != nil {
		t.Fatalf("failed to create money amount: %v", err)
	}

	// Create provider fee
	feeAmount, err := money.New(100, "USD") // 1.00 USD fee
	if err != nil {
		t.Fatalf("failed to create fee amount: %v", err)
	}

	// Create the event using the factory function with options
	event := events.NewPaymentCompleted(
		events.FlowEvent{
			ID:            testEventID,
			FlowType:      "payment",
			UserID:        testUserID,
			AccountID:     testAccountID,
			CorrelationID: testCorrelationID,
		},
		events.WithPaymentID(testPaymentID),
		events.WithProviderFee(account.Fee{
			Amount: feeAmount,
			Type:   account.FeeProvider,
		}),
	)

	// Set the transaction ID and amount
	event.TransactionID = testTransactionID
	event.Amount = amount

	validEvent := event

	t.Run("returns nil for incorrect event type", func(t *testing.T) {
		h := HandleCompleted(bus, mUow, logger)
		err := h(ctx, &mockEvent{})
		require.NoError(t, err)
	})

	t.Run("handles error from unit of work", func(t *testing.T) {
		h := HandleCompleted(bus, mUow, logger)
		mUow.On("Do", ctx, mock.Anything).Return(errors.New("uow error")).Once()
		err := h(ctx, validEvent)
		assert.Error(t, err)
	})

	t.Run("handles successful event", func(t *testing.T) {
		h := HandleCompleted(bus, mUow, logger)

		// Create a proper transaction with currency
		tx := &dto.TransactionRead{
			ID:        testTransactionID,
			UserID:    testUserID,
			AccountID: testAccountID,
			PaymentID: testPaymentID,
			Status:    string(account.TransactionStatusPending),
			Currency:  "USD",
			Amount:    100.0,
		}

		// Mock the repositories
		mockAccRepo := mocks.NewAccountRepository(t)
		mockTxRepo := mocks.NewTransactionRepository(t)

		// Mock account repository calls
		mockAccRepo.On("Get", ctx, testAccountID).Return(&dto.AccountRead{
			ID:       testAccountID,
			UserID:   testUserID,
			Balance:  100.0, // Initial balance
			Currency: "USD",
		}, nil)

		// Mock account update with a more flexible matcher
		mockAccRepo.On("Update", ctx, testAccountID, mock.MatchedBy(
			func(update dto.AccountUpdate) bool {
				// Accept any non-nil balance update
				return update.Balance != nil
			})).Return(nil).Once()

		// Mock transaction repository calls
		mockTxRepo.On("GetByPaymentID", ctx, testPaymentID).Return(tx, nil)
		mockTxRepo.On("Update", ctx, testTransactionID, mock.MatchedBy(
			func(update dto.TransactionUpdate) bool {
				return *update.Status == string(account.TransactionStatusCompleted)
			})).Return(nil)

		// Setup UOW mocks - match the exact number of calls in the handler
		// The handler makes 1 call to GetRepository for each repository type
		mUow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(mockAccRepo, nil).Once()
		mUow.On("GetRepository", (*repotransaction.Repository)(nil)).Return(mockTxRepo, nil).Once()

		// Mock the Do callback
		mUow.On("Do", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			// The callback that updates the transaction and account
			cb, ok := args[1].(func(repository.UnitOfWork) error)
			if !ok {
				t.Fatal("expected callback function")
			}

			err := cb(mUow)
			require.NoError(t, err)
		}).Once()

		err := h(ctx, validEvent)
		require.NoError(t, err)

		// Verify all expectations were met
		mockAccRepo.AssertExpectations(t)
		mockTxRepo.AssertExpectations(t)
		bus.AssertExpectations(t)
	})
}
