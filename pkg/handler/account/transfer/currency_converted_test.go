package transfer

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandleCurrencyConverted_Success(t *testing.T) {
	// Setup test data
	ctx := context.Background()
	logger := slog.Default()
	userID := uuid.New()
	sourceAccountID := uuid.New()
	destAccountID := uuid.New()
	transactionID := uuid.New()
	correlationID := uuid.New()

	// Create money amount
	amount, err := money.New(100.0, money.USD) // $100.00
	require.NoError(t, err)

	// Create TransferRequested event
	tr := &events.TransferRequested{
		FlowEvent: events.FlowEvent{
			ID:            uuid.New(),
			UserID:        userID,
			AccountID:     sourceAccountID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		DestAccountID: destAccountID,
		Amount:        amount,
	}

	// Create CurrencyConversionRequested with same currency for test simplicity
	ccr := &events.CurrencyConversionRequested{
		FlowEvent: events.FlowEvent{
			ID:            uuid.New(),
			UserID:        userID,
			AccountID:     sourceAccountID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		OriginalRequest: tr,
		Amount:          amount,
		To:              money.USD, // Using currency.USD instead of money.USD
		TransactionID:   transactionID,
	}

	// Create CurrencyConverted with same amount and currency as source
	convertedAmount, err := money.New(100.0, money.USD) // $100.00
	require.NoError(t, err)

	cc := &events.CurrencyConverted{
		CurrencyConversionRequested: *ccr,
		TransactionID:               transactionID,
		ConvertedAmount:             convertedAmount,
	}

	// Create TransferCurrencyConverted
	tcc := events.NewTransferCurrencyConverted(cc)

	// Create mocks
	bus := mocks.NewBus(t)
	uow := mocks.NewUnitOfWork(t)
	accRepo := mocks.NewAccountRepository(t)

	// Mock UoW to return account repository - called once for both source and destination accounts
	uow.On("GetRepository", (*account.Repository)(nil)).
		Return(accRepo, nil).
		Once()

	// Create a source account with sufficient balance
	sourceAcc := &dto.AccountRead{
		ID:        sourceAccountID,
		UserID:    userID,
		Balance:   1000.0, // $1000.00
		Currency:  "USD",
		CreatedAt: time.Now(),
	}

	// Create a destination account with the same currency for test simplicity
	destAcc := &dto.AccountRead{
		ID:        destAccountID,
		UserID:    userID,
		Balance:   500.0, // $500.00
		Currency:  "USD",
		CreatedAt: time.Now(),
	}

	// Mock Get for source account
	accRepo.On("Get", ctx, sourceAccountID).
		Return(sourceAcc, nil).
		Once()

	// Mock Get for destination account (needed for validation)
	accRepo.On("Get", ctx, destAccountID).
		Return(destAcc, nil).
		Once()

	// Mock bus to expect TransferValidated event with any payload
	bus.On("Emit", ctx, mock.MatchedBy(func(e events.Event) bool {
		_, ok := e.(*events.TransferValidated)
		return ok
	})).
		Return(nil).
		Once()

	// Execute
	handler := HandleCurrencyConverted(bus, uow, logger)
	err = handler(ctx, tcc)

	// Verify
	require.NoError(t, err)
	bus.AssertExpectations(t)
	uow.AssertExpectations(t)
	accRepo.AssertExpectations(t)
	// Verify the exact number of calls we expect
	uow.AssertNumberOfCalls(t, "GetRepository", 1) // Called once to get the account repository
	uow.AssertExpectations(t)                      // Ensure all UoW expectations were met
}
