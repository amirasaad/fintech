package withdraw_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/account/withdraw"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBusinessValidation(t *testing.T) {
	// Setup test logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	// Common test data
	var userID uuid.UUID
	var accountID uuid.UUID
	var transactionID uuid.UUID
	var correlationID uuid.UUID
	var validAmount money.Money
	var err error

	t.Run("successfully validates and emits payment initiation event", func(t *testing.T) {
		// Create a valid amount in USD
		amount, err := money.New(10000, "USD") // $100.00 in cents
		require.NoError(t, err)

		// Setup mocks
		mockUoW := new(mocks.MockUnitOfWork)
		mockAccRepo := new(mocks.AccountRepository)
		mockEventBus := mocks.NewMockEventBus(t)

		// Create business validation event
		event := &events.WithdrawBusinessValidationEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "withdraw",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			Amount: amount,
		}

		// Mock expectations
		mockUoW.On("GetRepository", (*repoaccount.Repository)(nil)).Return(mockAccRepo, nil)

		// Create a test account with sufficient balance
		testAccount := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  20000, // $200.00 in cents
			Currency: "USD",
		}
		mockAccRepo.On("Get", mock.Anything, accountID).Return(testAccount, nil)

		// Expect the payment initiation event to be emitted
		mockEventBus.On("Emit", mock.Anything, mock.MatchedBy(func(e events.PaymentInitiationEvent) bool {
			return e.TransactionID == transactionID &&
				e.Account.ID == accountID &&
				e.Amount.Amount() == 10000 &&
				e.Amount.Currency() == "USD"
		})).Return(nil)

		// Create handler and process event
		handler := withdraw.BusinessValidation(mockEventBus, mockUoW, logger)
		err = handler(ctx, event)

		// Assertions
		require.NoError(t, err)
		mockUoW.AssertExpectations(t)
		mockAccRepo.AssertExpectations(t)
		mockEventBus.AssertExpectations(t)
	})
	t.Run("successfully validates and emits payment initiation event", func(t *testing.T) {
		// Setup test data
		// Common test data
		var userID uuid.UUID
		var accountID uuid.UUID
		var transactionID uuid.UUID
		var correlationID uuid.UUID
		var validAmount money.Money

		// Create business validation event with all required fields
		event := events.NewWithdrawBusinessValidationEvent(
			userID,
			accountID,
			correlationID,
			func(bv *events.WithdrawBusinessValidationEvent) {
				// Set the required fields

				bv.Amount = validAmount
				// Initialize the embedded WithdrawValidatedEvent
				bv.WithdrawValidatedEvent = events.WithdrawValidatedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "withdraw",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: correlationID,
					},
					TargetCurrency: "USD",
				}
				// Set the conversion done event
				bv.ConversionDoneEvent = events.ConversionDoneEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "withdraw",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: correlationID,
					},
					TransactionID:   transactionID,
					ConvertedAmount: validAmount,
				}
			},
		)

		// Setup mocks
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		eventBus := mocks.NewEventBus(t)

		// Expectations
		uow.EXPECT().GetRepository((*account.Repository)(nil)).Return(accRepo, nil)
		accRepo.EXPECT().Get(mock.Anything, accountID).Return(&dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  decimal.NewFromInt(20000), // $200.00
			Currency: "USD",
		}, nil)

		eventBus.EXPECT().Emit(mock.Anything, mock.MatchedBy(func(e events.PaymentInitiationEvent) bool {
			return e.TransactionID == transactionID &&
				e.Amount.Equals(validAmount) &&
				e.Account.ID == accountID
		})).Return(nil)

		// Execute
		handler := withdraw.BusinessValidation(eventBus, uow, slog.Default())
		err := handler(context.Background(), event)

		// Verify
		require.NoError(t, err)
	})

	t.Run("returns error for insufficient funds", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		// Create test event
		// Create flow event
		flowEvent := events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
			FlowType:      "withdraw",
		}

		// Create business validation event with currency
		event := events.NewWithdrawBusinessValidationEvent(
			userID,
			accountID,
			correlationID,
			func(bv *events.WithdrawBusinessValidationEvent) {
				// Set the amount with currency
				bv.Amount = validAmount
				// Set the conversion done event
				bv.ConversionDoneEvent = events.ConversionDoneEvent{
					FlowEvent:       flowEvent,
					TransactionID:   transactionID,
					ConvertedAmount: validAmount,
				}
				// Set the target currency in the WithdrawValidatedEvent
				bv.WithdrawValidatedEvent.TargetCurrency = "USD"
			},
		)

		// Create account with insufficient balance (balance in cents)
		accDto := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  50, // $0.50 in cents (less than the $1.00 withdrawal)
			Currency: "USD",
		}

		// Set up mocks
		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accDto, nil).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Should return an error about insufficient funds
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient funds")
		uow.AssertExpectations(t)
		accRepo.AssertExpectations(t)
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		// Create test event
		// Create flow event
		flowEvent := events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
			FlowType:      "withdraw",
		}

		// Create business validation event with currency
		event := events.NewWithdrawBusinessValidationEvent(
			userID,
			accountID,
			correlationID,
			func(bv *events.WithdrawBusinessValidationEvent) {
				// Set the amount with currency
				bv.Amount = validAmount
				// Set the conversion done event
				bv.ConversionDoneEvent = events.ConversionDoneEvent{
					FlowEvent:       flowEvent,
					TransactionID:   transactionID,
					ConvertedAmount: validAmount,
				}
				// Set the target currency in the WithdrawValidatedEvent
				bv.WithdrawValidatedEvent.TargetCurrency = "USD"
			},
		)

		// Set up mocks to return account not found error
		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(nil, domain.ErrAccountNotFound).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
		uow.AssertExpectations(t)
		accRepo.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Create test event
		// Create flow event
		flowEvent := events.FlowEvent{
			UserID:        userID,
			AccountID:     accountID,
			CorrelationID: correlationID,
			FlowType:      "withdraw",
		}

		// Create business validation event with currency
		event := events.NewWithdrawBusinessValidationEvent(
			userID,
			accountID,
			correlationID,
			func(bv *events.WithdrawBusinessValidationEvent) {
				// Set the amount with currency
				bv.Amount = validAmount
				// Set the conversion done event
				bv.ConversionDoneEvent = events.ConversionDoneEvent{
					FlowEvent:       flowEvent,
					TransactionID:   transactionID,
					ConvertedAmount: validAmount,
				}
				// Set the target currency in the WithdrawValidatedEvent
				bv.WithdrawValidatedEvent.TargetCurrency = "USD"
			},
		)

		// Set up mocks to return error when getting repository
		expectedErr := assert.AnError
		uow.On("GetRepository", (*repoaccount.Repository)(nil)).Return(nil, expectedErr).Once()

		handler := withdraw.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.ErrorIs(t, err, expectedErr)
		uow.AssertExpectations(t)
	})
}
