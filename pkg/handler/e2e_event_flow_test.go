package handler_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	mocks "github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDepositE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	// Setup mocks
	accRepo := mocks.NewAccountRepository(t)
	uow := mocks.NewMockUnitOfWork(t)
	bus := eventbus.NewSimpleEventBus()

	// Prepare account read DTO
	accRead := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  amount.AmountFloat(),
		Currency: amount.Currency().String(),
	}

	// Expectations
	accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil)
	uow.On("GetRepository", mock.Anything).Return(accRepo, nil)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil)

	// Track event emissions
	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	// Register handlers (real logic)
	logger := slog.Default()
	bus.Register("DepositRequestedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositRequestedEvent")
		deposithandler.ValidationHandler(bus, uow, logger)(ctx, e) //nolint:errcheck
		return nil
	})
	bus.Register("DepositValidatedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositValidatedEvent")
		deposithandler.PersistenceHandler(bus, uow, logger)(ctx, e) //nolint:errcheck
		return nil
	})
	bus.Register("DepositPersistedEvent", func(ctx context.Context, e domain.Event) error {
		track("DepositPersistedEvent")
		// End of chain for this E2E
		return nil
	})

	// Start the chain
	_ = bus.Emit(ctx, events.DepositRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     accountID,
			UserID:        userID,
			CorrelationID: uuid.New(),
			FlowType:      "deposit",
		},
		Amount:    amount,
		Timestamp: time.Now(),
	})

	// Wait a moment for all handlers to run (since event bus is sync, this is immediate)
	time.Sleep(10 * time.Millisecond)

	// Assert event chain
	assert.Equal(t, []string{
		"DepositRequestedEvent",
		"DepositValidatedEvent",
		"DepositPersistedEvent",
	}, emitted, "event chain should match deposit flow")
}
