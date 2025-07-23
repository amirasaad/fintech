package deposit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully validates and emits payment initiation event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositBusinessValidationEvent{
			DepositValidatedEvent: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "deposit",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: uuid.New(),
					},
					Amount: amount,
				},
			},
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			paymentInitiationEvent, ok := e.(events.PaymentInitiationEvent)
			if !ok {
				return false
			}
			return paymentInitiationEvent.TransactionID == transactionID &&
				paymentInitiationEvent.FlowType == "deposit"
		})).Return(nil).Once()

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		// Use a different event type
		event := events.WithdrawBusinessValidationEvent{}

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// No interactions should occur with mocks
		uow.AssertNotCalled(t, "GetRepository", mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles repository error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositBusinessValidationEvent{
			DepositValidatedEvent: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "deposit",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: uuid.New(),
					},
					Amount: amount,
				},
			},
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			Amount: amount,
		}

		// Mock repository error
		uow.On("GetRepository", mock.Anything).Return(nil, errors.New("repository error")).Once()

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles invalid repository type", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositBusinessValidationEvent{
			DepositValidatedEvent: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "deposit",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: uuid.New(),
					},
					Amount: amount,
				},
			},
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			Amount: amount,
		}

		// Mock returning wrong repository type
		uow.On("GetRepository", mock.Anything).Return("wrong type", nil).Once()

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles account not found", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositBusinessValidationEvent{
			DepositValidatedEvent: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "deposit",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: uuid.New(),
					},
					Amount: amount,
				},
			},
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			Amount: amount,
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(nil, errors.New("account not found")).Once()

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles business validation failure", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		wrongUserID := uuid.New() // Different user ID to trigger validation failure
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.DepositBusinessValidationEvent{
			DepositValidatedEvent: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "deposit",
						UserID:        wrongUserID, // Wrong user ID
						AccountID:     accountID,
						CorrelationID: uuid.New(),
					},
					Amount: amount,
				},
			},
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        wrongUserID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID, // Account belongs to different user
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err) // Should return validation error
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles negative amount validation failure", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(-100, currency.USD) // Negative amount

		event := events.DepositBusinessValidationEvent{
			DepositValidatedEvent: events.DepositValidatedEvent{
				DepositRequestedEvent: events.DepositRequestedEvent{
					FlowEvent: events.FlowEvent{
						FlowType:      "deposit",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: uuid.New(),
					},
					Amount: amount,
				},
			},
			ConversionDoneEvent: events.ConversionDoneEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				TransactionID: transactionID,
			},
			Amount: amount,
		}

		accRead := &dto.AccountRead{
			ID:       accountID,
			UserID:   userID,
			Balance:  1000.0,
			Currency: "USD",
		}

		// Mock expectations
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil).Once()
		accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil).Once()

		// Execute
		handler := BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err) // Should return validation error for negative amount
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
