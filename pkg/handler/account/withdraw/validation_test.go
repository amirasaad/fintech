package withdraw

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWithdrawValidationHandler(t *testing.T) {
	validUserID := uuid.New()
	validAccountID := uuid.New()
	validEvent := events.WithdrawRequestedEvent{
		UserID:    validUserID,
		AccountID: validAccountID,
		Amount:    money.NewFromData(10000, "USD"),
	}
	invalidEvent := events.WithdrawRequestedEvent{
		UserID:    uuid.Nil,
		AccountID: uuid.Nil,
		Amount:    money.NewFromData(-5000, "USD"),
	}

	tests := []struct {
		name       string
		input      events.WithdrawRequestedEvent
		expectPub  bool
		setupMocks func(bus *mocks.MockEventBus, uow *mocks.MockUnitOfWork)
	}{
		{"valid event", validEvent, true, func(bus *mocks.MockEventBus, uow *mocks.MockUnitOfWork) {
			accountRepo := new(mocks.AccountRepository)
			uow.On("GetRepository", mock.Anything).Return(accountRepo, nil)
			accountRepo.On("Get", mock.Anything, validAccountID).Return(&dto.AccountRead{
				ID:       validAccountID,
				UserID:   validUserID,
				Balance:  10000,
				Currency: "USD",
			}, nil)
			bus.On("Publish", mock.Anything, mock.AnythingOfType("events.WithdrawValidatedEvent")).Return(nil)
		}},
		{"invalid event", invalidEvent, false, nil},
		{"invalid accountID (empty)", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: uuid.Nil, Amount: money.NewFromData(10000, "USD")}, false, nil},
		{"zero amount", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: money.NewFromData(0, "USD")}, false, nil},
		{"negative amount", events.WithdrawRequestedEvent{UserID: validUserID, AccountID: validAccountID, Amount: money.NewFromData(-1000, "USD")}, false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bus := mocks.NewMockEventBus(t)
			uow := mocks.NewMockUnitOfWork(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus, uow)
			}
			handler := WithdrawValidationHandler(bus, uow, slog.New(slog.NewTextHandler(io.Discard, nil)))
			ctx := context.Background()
			handler(ctx, tc.input)
			if tc.expectPub {
				assert.True(t, bus.AssertCalled(t, "Publish", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent")), "should publish WithdrawValidatedEvent")
			} else {
				bus.AssertNotCalled(t, "Publish", ctx, mock.AnythingOfType("events.WithdrawValidatedEvent"))
			}
		})
	}
}
