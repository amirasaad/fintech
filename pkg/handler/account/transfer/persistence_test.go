package transfer_test

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
	"github.com/amirasaad/fintech/pkg/handler/account/transfer"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully persists transaction and emits event", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		txRepo := mocks.NewTransactionRepository(t)
		accRepo := mocks.NewAccountRepository(t)

		amount, _ := money.New(100, currency.USD)
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		destAccountID := uuid.New()

		transferRequested := events.NewTransferRequestedEvent(
			userID, accountID, correlationID,
			events.WithTransferRequestedAmount(amount),
			events.WithTransferDestAccountID(destAccountID),
		)
		validated := events.NewTransferValidatedEvent(
			userID, accountID, correlationID,
			events.WithTransferRequestedEvent(*transferRequested),
		)
		conversionDone := events.NewConversionDoneEvent(
			userID,
			accountID,
			correlationID,
			events.WithConvertedAmount(amount),
		)
		event := events.NewTransferDomainOpDoneEvent(
			events.WithTransferFlowEvent(validated.FlowEvent),
			events.WithTransferAmount(amount),
			events.WithDestAccountID(destAccountID),
			func(e *events.TransferDomainOpDoneEvent) {
				e.TransferValidatedEvent = *validated
				e.ConversionDoneEvent = conversionDone
			},
		)

		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
			Return(nil).
			Run(func(args mock.Arguments) {
				fn := args.Get(1).(func(repository.UnitOfWork) error)
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isTx := repoType.(*transaction.Repository)
					return isTx
				})).Return(txRepo, nil)
				uow.On("GetRepository", mock.MatchedBy(func(repoType interface{}) bool {
					_, isAcc := repoType.(*account.Repository)
					return isAcc
				})).Return(accRepo, nil)
				accRepo.On("Update", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.AccountUpdate")).Return(nil)
				txRepo.On("Update", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.TransactionUpdate")).Return(nil)
				assert.NoError(t, fn(uow))
			})

		bus.On("Emit", mock.Anything, mock.AnythingOfType("*events.TransferCompletedEvent")).Return(nil).Once()

		handler := transfer.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		assert.NoError(t, err)
	})

	t.Run("fails gracefully when repository error occurs", func(t *testing.T) {
		bus := mocks.NewMockBus(t)
		uow := mocks.NewMockUnitOfWork(t)
		accRepo := mocks.NewAccountRepository(t)

		amount, _ := money.New(100, currency.USD)
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		destAccountID := uuid.New()

		transferRequested := events.NewTransferRequestedEvent(
			userID, accountID, correlationID,
			events.WithTransferRequestedAmount(amount),
			events.WithTransferDestAccountID(destAccountID),
		)
		validated := events.NewTransferValidatedEvent(
			userID, accountID, correlationID,
			events.WithTransferRequestedEvent(*transferRequested),
		)
		conversionDone := events.NewConversionDoneEvent(
			userID,
			accountID,
			correlationID,
			events.WithConvertedAmount(amount),
		)
		event := events.NewTransferDomainOpDoneEvent(
			events.WithTransferFlowEvent(validated.FlowEvent),
			events.WithTransferAmount(amount),
			events.WithDestAccountID(destAccountID),
			func(e *events.TransferDomainOpDoneEvent) {
				e.TransferValidatedEvent = *validated
				e.ConversionDoneEvent = conversionDone
			},
		)

		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(errors.New("repository error"))
		uow.On("GetRepository", mock.Anything).Return(accRepo, nil)

		handler := transfer.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		assert.Error(t, err)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		uow := mocks.NewMockUnitOfWork(t)
		bus := mocks.NewMockBus(t)

		amount, _ := money.New(100, currency.USD)
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		destAccountID := uuid.New()

		transferRequested := events.NewTransferRequestedEvent(
			userID, accountID, correlationID,
			events.WithTransferRequestedAmount(amount),
			events.WithTransferDestAccountID(destAccountID),
		)
		validated := events.NewTransferValidatedEvent(
			userID, accountID, correlationID,
			events.WithTransferRequestedEvent(*transferRequested),
		)
		conversionDone := events.NewConversionDoneEvent(
			userID,
			accountID,
			correlationID,
			events.WithConvertedAmount(amount),
		)
		event := events.NewTransferDomainOpDoneEvent(
			events.WithTransferFlowEvent(validated.FlowEvent),
			events.WithTransferAmount(amount),
			events.WithDestAccountID(destAccountID),
			func(e *events.TransferDomainOpDoneEvent) {
				e.TransferValidatedEvent = *validated
				e.ConversionDoneEvent = conversionDone
			},
		)
		dbError := errors.New("database error")
		uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil)

		uow.On("GetRepository", mock.Anything).Return(nil, dbError).Once()

		handler := transfer.Persistence(bus, uow, logger)
		err := handler(ctx, event)

		assert.ErrorIs(t, err, dbError)
	})
}
