package payment_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account"
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
	txRepo := mocks.NewMockTransactionRepository(t)

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	tx := &dto.TransactionRead{
		ID: event.TransactionID,
	}

	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		fn(uow)
	})
	uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Maybe()
	txRepo.On("Get", mock.Anything, event.TransactionID).Return(tx, nil)
	status := string(account.TransactionStatusPending)
	txRepo.On("Update", mock.Anything, event.TransactionID, dto.TransactionUpdate{
		PaymentID: &event.PaymentID,
		Status:    &status,
	}).Return(nil)

	err := handler(context.Background(), event)

	assert.NoError(t, err)
	uow.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestPersistence_DuplicateEvent(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)
	txRepo := mocks.NewMockTransactionRepository(t)

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	tx := &dto.TransactionRead{
		ID:        event.TransactionID,
		PaymentID: "pm_existing",
	}

	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		fn(uow)
	})
	uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Maybe()
	txRepo.On("Get", mock.Anything, event.TransactionID).Return(tx, nil)

	err := handler(context.Background(), event)

	assert.NoError(t, err)
	txRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
	uow.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestPersistence_GetTransactionError(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)
	txRepo := mocks.NewMockTransactionRepository(t)

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		fn(uow)
	})
	uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Maybe()
	txRepo.On("Get", mock.Anything, event.TransactionID).Return(nil, errors.New("get error"))

	err := handler(context.Background(), event)

	assert.NoError(t, err)
	uow.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestPersistence_UpdateTransactionError(t *testing.T) {
	logger := slog.Default()
	bus := mocks.NewMockBus(t)
	uow := mocks.NewMockUnitOfWork(t)
	txRepo := mocks.NewMockTransactionRepository(t)

	handler := payment.Persistence(bus, uow, logger)

	event := events.PaymentInitiatedEvent{
		TransactionID: uuid.New(),
		PaymentID:     "pm_12345",
	}

	tx := &dto.TransactionRead{
		ID: event.TransactionID,
	}

	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		fn(uow)
	})
	uow.On("GetRepository", (*transaction.Repository)(nil)).Return(txRepo, nil).Maybe()
	txRepo.On("Get", mock.Anything, event.TransactionID).Return(tx, nil)
	status := string(account.TransactionStatusPending)
	txRepo.On("Update", mock.Anything, event.TransactionID, dto.TransactionUpdate{
		PaymentID: &event.PaymentID,
		Status:    &status,
	}).Return(errors.New("update error"))

	err := handler(context.Background(), event)

	assert.NoError(t, err)
	uow.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}