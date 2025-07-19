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
	"github.com/amirasaad/fintech/pkg/eventbus"
	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDepositE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	// Setup mocks
	// accRepo := mocks.NewAccountRepository(t)
	// txRepo := mocks.NewTransactionRepository(t)
	uow := mocks.NewMockUnitOfWork(t)
	bus := eventbus.NewSimpleEventBus()



	// Prepare account read DTO
	// accRead := &dto.AccountRead{
	// 	ID:       accountID,
	// 	UserID:   userID,
	// 	Balance:  amount.AmountFloat(),
	// 	Currency: amount.Currency().String(),
	// }

	// Expectations
	// accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil)
	// txRepo.On("Create", mock.Anything, mock.AnythingOfType("dto.TransactionCreate")).Return(nil)

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
	bus.Subscribe("DepositRequestedEvent", func(ctx context.Context, e domain.Event) {
		track("DepositRequestedEvent")
		deposithandler.ValidationHandler(bus, uow, logger)(ctx, e)
	})
	bus.Subscribe("DepositValidatedEvent", func(ctx context.Context, e domain.Event) {
		track("DepositValidatedEvent")
		deposithandler.PersistenceHandler(bus, uow, logger)(ctx, e)
	})
	bus.Subscribe("DepositPersistedEvent", func(ctx context.Context, e domain.Event) {
		track("DepositPersistedEvent")
		// End of chain for this E2E
	})

	// Start the chain
	bus.Publish(ctx, events.DepositRequestedEvent{
		EventID:   uuid.New(),
		AccountID: accountID,
		UserID:    userID,
		Amount:    amount,
		Source:    "deposit",
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
