package withdraw

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestWithdrawValidationHandler(t *testing.T) {
	validUserID := uuid.New()
	validAccountID := uuid.New()
	validEvent := events.WithdrawRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     validAccountID,
			UserID:        validUserID,
			CorrelationID: uuid.New(),
			FlowType:      "withdraw",
		},
		Amount: money.NewFromData(10000, "USD"),
	}
	invalidEvent := events.WithdrawRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     uuid.Nil,
			UserID:        uuid.Nil,
			CorrelationID: uuid.New(),
			FlowType:      "withdraw",
		},
		Amount: money.NewFromData(-5000, "USD"),
	}

	tests := []struct {
		name       string
		input      events.WithdrawRequestedEvent
		expectPub  bool
		setupMocks func(bus *mocks.MockBus, uow *mocks.MockUnitOfWork)
	}{
		{"valid event", validEvent, true, func(bus *mocks.MockBus, uow *mocks.MockUnitOfWork) {
			accountRepo := new(mocks.AccountRepository)
			uow.On("GetRepository", mock.Anything).Return(accountRepo, nil)
			accountRepo.On("Get", mock.Anything, validAccountID).Return(&dto.AccountRead{
				ID:       validAccountID,
				UserID:   validUserID,
				Balance:  10000,
				Currency: "USD",
			}, nil)
			bus.On("Emit", mock.Anything, mock.AnythingOfType("events.WithdrawValidatedEvent")).Return(nil)
		}},
		{"invalid event", invalidEvent, false, func(bus *mocks.MockBus, uow *mocks.MockUnitOfWork) {
			// Mock GetRepository for invalid event (handler calls it before validation)
			accountRepo := new(mocks.AccountRepository)
			uow.On("GetRepository", mock.Anything).Return(accountRepo, nil)
			accountRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("account not found"))
		}},
		{"invalid accountID (empty)", events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID:     uuid.Nil,
				UserID:        validUserID,
				CorrelationID: uuid.New(),
				FlowType:      "withdraw",
			},
			Amount: money.NewFromData(10000, "USD"),
		}, false, func(bus *mocks.MockBus, uow *mocks.MockUnitOfWork) {
			// Mock GetRepository for invalid account ID
			accountRepo := new(mocks.AccountRepository)
			uow.On("GetRepository", mock.Anything).Return(accountRepo, nil)
			accountRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("account not found"))
		}},
		{"zero amount", events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID:     validAccountID,
				UserID:        validUserID,
				CorrelationID: uuid.New(),
				FlowType:      "withdraw",
			},
			Amount: money.NewFromData(0, "USD"),
		}, false, func(bus *mocks.MockBus, uow *mocks.MockUnitOfWork) {
			// Mock GetRepository for zero amount (handler calls it before validation)
			accountRepo := new(mocks.AccountRepository)
			uow.On("GetRepository", mock.Anything).Return(accountRepo, nil)
			accountRepo.On("Get", mock.Anything, validAccountID).Return(&dto.AccountRead{
				ID:       validAccountID,
				UserID:   validUserID,
				Balance:  10000,
				Currency: "USD",
			}, nil)
		}},
		{"negative amount", events.WithdrawRequestedEvent{
			FlowEvent: events.FlowEvent{
				AccountID:     validAccountID,
				UserID:        validUserID,
				CorrelationID: uuid.New(),
				FlowType:      "withdraw",
			},
			Amount: money.NewFromData(-1000, "USD"),
		}, false, func(bus *mocks.MockBus, uow *mocks.MockUnitOfWork) {
			// Mock GetRepository for negative amount (handler calls it before validation)
			accountRepo := new(mocks.AccountRepository)
			uow.On("GetRepository", mock.Anything).Return(accountRepo, nil)
			accountRepo.On("Get", mock.Anything, validAccountID).Return(&dto.AccountRead{
				ID:       validAccountID,
				UserID:   validUserID,
				Balance:  10000,
				Currency: "USD",
			}, nil)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockBus(t)
			uow := mocks.NewMockUnitOfWork(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus, uow)
			}
			handler := WithdrawValidationHandler(bus, uow, slog.New(slog.NewTextHandler(io.Discard, nil)))
			ctx := context.Background()
			handler(ctx, tc.input) //nolint:errcheck
			if tc.expectPub {
				bus.AssertCalled(t, "Emit", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent"))
			} else {
				bus.AssertNotCalled(t, "Emit", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent"))
			}
		})
	}
}
