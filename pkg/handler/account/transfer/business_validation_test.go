package transfer_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	transfer "github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestBusinessValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully validates and emits domain op done event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)
		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Use the helper to create a fully populated event
		event := NewValidTransferBusinessValidationEvent(userID, accountID, correlationID, destAccountID, amount)

		sufficientBalance, _ := money.New(200, currency.USD)

		uow.On("GetRepository", (*account.Repository)(nil)).Return(accRepo, nil).Once()
		// Create a properly initialized account with all required fields
		testAccount := &dto.AccountRead{
			ID:        accountID,
			UserID:    userID,
			Balance:   sufficientBalance.AmountFloat(),
			Currency:  "USD",
			CreatedAt: time.Now(),
		}
		// Create a strict matcher for context that also checks for timeout
		ctxMatcher := mock.MatchedBy(func(ctx context.Context) bool {
			// Check if context is not nil and has a deadline set
			if ctx == nil {
				t.Log("Context is nil")
				return false
			}
			// Verify the context has a deadline (which should be set by the handler)
			_, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				t.Log("Context is missing deadline")
			}
			return hasDeadline
		})

		// Match the exact account ID with strict UUID comparison
		accRepo.On("Get", ctxMatcher, accountID).Return(testAccount, nil).Once()

		done := make(chan bool, 1)
		// Use mock.MatchedBy to match the event structure
		// Match both context and event with proper type checking
		bus.On("Emit", ctxMatcher, mock.MatchedBy(func(e interface{}) bool {
			switch evt := e.(type) {
			case *events.TransferDomainOpDoneEvent:
				// Strictly validate all UUIDs match expected values
				isValid := evt.UserID == userID &&
					evt.AccountID == accountID &&
					evt.CorrelationID == correlationID &&
					evt.DestAccountID == destAccountID &&
					evt.TransactionID == event.TransactionID &&
					evt.TransactionID != uuid.Nil &&
					evt.AccountID != uuid.Nil &&
					evt.UserID != uuid.Nil &&
					evt.CorrelationID != uuid.Nil
				if isValid {
					t.Logf("Matched TransferDomainOpDoneEvent: %+v", evt)
					go func() { done <- true }()
				}
				return isValid
			case *events.TransferFailedEvent:
				t.Logf("Unexpected TransferFailedEvent: %+v", evt)
				go func() { done <- false }()
				return false
			default:
				t.Logf("Unexpected event type: %T", e)
				go func() { done <- false }()
				return false
			}
		})).Return(nil).Once()

		handler := transfer.BusinessValidation(bus, uow, logger)
		err := handler(ctx, event)

		assert.NoError(t, err)
		bus.AssertExpectations(t)
		accRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("emits failed event for insufficient funds", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := NewValidTransferBusinessValidationEvent(userID, accountID, correlationID, destAccountID, amount)

		// Setup mocks
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		// Setup test account with insufficient funds (50 when we need 100)
		insufficientBalance, _ := money.New(50, currency.USD)

		uow.On("GetRepository", (*account.Repository)(nil)).Return(accRepo, nil).Once()

		testAccount := &dto.AccountRead{
			ID:        accountID,
			UserID:    userID,
			Balance:   insufficientBalance.AmountFloat(),
			Currency:  "USD",
			CreatedAt: time.Now(),
		}

		accRepo.On("Get", mock.Anything, accountID).Return(testAccount, nil).Once()

		done := make(chan bool, 1)
		bus.On("Emit", mock.Anything, mock.MatchedBy(func(e interface{}) bool {
			switch evt := e.(type) {
			case *events.TransferFailedEvent:
				isValid := evt.UserID == userID &&
					evt.AccountID == accountID &&
					evt.CorrelationID == correlationID &&
					evt.Reason == "insufficient funds"
				t.Logf("Matched TransferFailedEvent: %+v", evt)
				done <- isValid
				return isValid
			default:
				t.Logf("Unexpected event type in insufficient funds test: %T", e)
				done <- false
				return false
			}
		})).Return(nil).Once()

		handler := transfer.BusinessValidation(bus, uow, logger)

		// Execute
		err := handler(context.Background(), event)
		require.NoError(t, err)

		// Wait for the event to be processed with a timeout
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()

		select {
		case success := <-done:
			if !success {
				t.Fatal("Unexpected event received")
			}
		case <-timer.C:
			// Get all mock calls for debugging
			for _, call := range bus.Calls {
				t.Logf("Bus call: %+v", call.Arguments)
			}
			t.Fatal("Timed out waiting for TransferFailedEvent")
		}

		// Verify
		bus.AssertExpectations(t)
		uow.AssertExpectations(t)
		accRepo.AssertExpectations(t)
	})

	t.Run("emits failed event when account not found", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Create the event first so we can reference its fields
		// Create the event with all required fields
		event := NewValidTransferBusinessValidationEvent(userID, accountID, correlationID, destAccountID, amount)

		// Ensure all UUIDs are properly set in the event hierarchy
		event.UserID = userID
		event.AccountID = accountID
		event.CorrelationID = correlationID

		// Set the same UUIDs in the nested events
		event.TransferValidatedEvent.UserID = userID
		event.TransferValidatedEvent.AccountID = accountID
		event.TransferValidatedEvent.CorrelationID = correlationID

		event.TransferRequestedEvent.UserID = userID
		event.TransferRequestedEvent.AccountID = accountID
		event.TransferRequestedEvent.CorrelationID = correlationID

		// Setup mocks
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		// Setup expectations
		// Use mock.MatchedBy to properly match the context argument
		ctxMatcher := mock.MatchedBy(func(ctx context.Context) bool {
			// We can add more specific context validation here if needed
			return ctx != nil
		})

		// Setup repository expectations
		uow.On("GetRepository", (*account.Repository)(nil)).Return(accRepo, nil).Once()
		accRepo.On("Get", ctxMatcher, accountID).Return((*dto.AccountRead)(nil), domain.ErrAccountNotFound).Once()

		// Setup event bus expectation with proper argument matching
		done := make(chan bool, 1)
		bus.On("Emit", ctxMatcher, mock.MatchedBy(func(e interface{}) bool {
			tfe, ok := e.(*events.TransferFailedEvent)
			if !ok {
				t.Logf("Expected *events.TransferFailedEvent, got: %T", e)
				done <- false
				return false
			}
			// Verify the event has the expected structure
			isValid := tfe.UserID == userID &&
				tfe.AccountID == accountID &&
				tfe.CorrelationID == correlationID &&
				tfe.Reason == "source account not found"
			if !isValid {
				t.Logf("Unexpected event data: %+v", tfe)
				done <- false
				return false
			}
			t.Logf("Matched TransferFailedEvent: %+v", tfe)
			done <- true
			return true
		})).Return(nil).Once()

		handler := transfer.BusinessValidation(bus, uow, logger)
		err := handler(context.Background(), event)

		// Wait for the event to be processed with a timeout
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()

		select {
		case success := <-done:
			if !success {
				t.Fatal("Unexpected event received")
			}
		case <-timer.C:
			t.Fatal("Timed out waiting for TransferFailedEvent")
		}

		assert.NoError(t, err)
		bus.AssertExpectations(t)
		accRepo.AssertExpectations(t)
		uow.AssertExpectations(t)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		uow := mocks.NewMockUnitOfWork(t)
		bus := mocks.NewMockBus(t)
		errMsg := "database error"

		// Expect GetRepository to be called and return an error
		uow.On("GetRepository", (*account.Repository)(nil)).Return(nil, errors.New(errMsg)).Once()

		handler := transfer.BusinessValidation(bus, uow, logger)

		// Create a valid event
		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)
		event := NewValidTransferBusinessValidationEvent(userID, accountID, correlationID, destAccountID, amount)

		err := handler(context.Background(), event)

		// Verify the error is returned and mocks are called as expected
		assert.ErrorContains(t, err, errMsg)
		uow.AssertExpectations(t)
	})

	t.Run("returns error on repository error", func(t *testing.T) {
		// Setup test data
		errMsg := "database error"
		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		// Setup mocks
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		// Setup context matcher
		ctxMatcher := mock.MatchedBy(func(ctx context.Context) bool {
			// We can add more specific context validation here if needed
			return ctx != nil
		})

		// Setup repository expectations
		uow.On("GetRepository", (*account.Repository)(nil)).Return(accRepo, nil).Once()
		accRepo.On("Get", ctxMatcher, accountID).Return((*dto.AccountRead)(nil), errors.New(errMsg)).Once()

		// Setup event bus expectation with proper argument matching
		done := make(chan bool, 1)
		bus.On("Emit", ctxMatcher, mock.MatchedBy(func(e interface{}) bool {
			tfe, ok := e.(*events.TransferFailedEvent)
			if !ok {
				t.Logf("Expected *events.TransferFailedEvent, got: %T", e)
				done <- false
				return false
			}
			// Verify the event contains the expected data
			if tfe.TransferRequestedEvent.ID == (uuid.UUID{}) ||
				tfe.TransferRequestedEvent.UserID != userID ||
				tfe.TransferRequestedEvent.AccountID != accountID ||
				tfe.TransferRequestedEvent.DestAccountID != destAccountID ||
				tfe.TransferRequestedEvent.CorrelationID != correlationID ||
				tfe.TransferRequestedEvent.Amount != amount ||
				tfe.Reason != "failed to get account: "+errMsg {
				t.Logf("Unexpected event data: %+v", tfe)
				done <- false
				return false
			}
			t.Logf("Matched TransferFailedEvent: %+v", tfe)
			done <- true
			return true
		})).Return(nil).Once()

		handler := transfer.BusinessValidation(bus, uow, logger)

		event := NewValidTransferBusinessValidationEvent(userID, accountID, correlationID, destAccountID, amount)

		err := handler(context.Background(), event)

		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()

		select {
		case success := <-done:
			if !success {
				t.Fatal("Unexpected event received")
			}
		case <-timer.C:
			t.Fatal("Timed out waiting for TransferFailedEvent")
		}

		assert.ErrorContains(t, err, errMsg)
		bus.AssertExpectations(t)
		uow.AssertExpectations(t)
		accRepo.AssertExpectations(t)
	})
}
