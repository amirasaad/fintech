package payment_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/repository"

	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPersistence(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(nil).Once()

	err := handler(context.Background(), event)

	assert.NoError(t, err)
}

func TestPersistence_ExistingPaymentID(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)
	var txRepo *mocks.TransactionRepository

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	tx := &dto.TransactionRead{
		ID:        event.TransactionID,
		PaymentID: "pm_existing",
	}

	uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(errors.New("update error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		txRepo = mocks.NewTransactionRepository(t)
		uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Once()
		txRepo.On("Get", mock.Anything, event.TransactionID).Return(tx, nil).Once()
		fn(uow) //nolint:errcheck
	}).Once()

	err := handler(context.Background(), event)

	assert.Error(t, err)
}

func TestPersistence_GetTransactionError(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)
	var txRepo *mocks.TransactionRepository

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(errors.New("update error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		txRepo = mocks.NewTransactionRepository(t)
		uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Once()
		txRepo.On("Get", mock.Anything, event.TransactionID).Return(nil, errors.New("get error")).Once()
		fn(uow) //nolint:errcheck
	}).Once()

	err := handler(context.Background(), event)

	assert.Error(t, err)
	uow.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestPersistence_UpdateTransactionError(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)
	var txRepo *mocks.TransactionRepository

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	uow.On("Do", mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).Return(errors.New("update error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		txRepo = mocks.NewTransactionRepository(t)
		uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Once()
		txRepo.On("Get", mock.Anything, event.TransactionID).Return(&dto.TransactionRead{}, nil).Once()
		txRepo.On("Update", mock.Anything, event.TransactionID, mock.AnythingOfType("dto.TransactionUpdate")).Return(errors.New("update error")).Once()
		_ = fn(uow) // Call the function, but the error is handled by the outer mock
	}).Once()

	err := handler(context.Background(), event)

	assert.Error(t, err)
}
