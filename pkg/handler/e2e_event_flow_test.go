package handler_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/amirasaad/fintech/pkg/dto"
	mocks "github.com/amirasaad/fintech/internal/fixtures/mocks"
)

type inMemoryAccountRepo struct {
	accounts map[uuid.UUID]*account.Account
	mu       sync.RWMutex
}

func (r *inMemoryAccountRepo) Get(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	acc, ok := r.accounts[id]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return acc, nil
}

func (r *inMemoryAccountRepo) Save(ctx context.Context, acc *account.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[acc.ID] = acc
	return nil
}

// Implement other methods as needed (stubbed for E2E)

// inMemoryUow implements repository.UnitOfWork
// Only GetRepository is used in handlers for E2E

type inMemoryUow struct {
	accRepo *inMemoryAccountRepo
}

func (u *inMemoryUow) GetRepository(repoType interface{}) (interface{}, error) {
	switch repoType.(type) {
	case *account.Repository:
		return u.accRepo, nil
	}
	return nil, repository.ErrRepositoryNotFound
}

func (u *inMemoryUow) Do(ctx context.Context, fn func(repository.UnitOfWork) error) error {
	return fn(u)
}

type mockUow struct {
	accRepo *mocks.AccountRepository
	txRepo  *mocks.TransactionRepository
}

func (u *mockUow) GetRepository(repoType interface{}) (interface{}, error) {
	switch repoType.(type) {
	case *account.Repository:
		return u.accRepo, nil
	case *transaction.Repository:
		return u.txRepo, nil
	}
	return nil, repository.ErrRepositoryNotFound
}

func (u *mockUow) Do(ctx context.Context, fn func(repository.UnitOfWork) error) error {
	return fn(u)
}

func TestDepositE2EEventFlow(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	// Setup mocks
	accRepo := mocks.NewAccountRepository(t)
	txRepo := mocks.NewTransactionRepository(t)
	uow := &mockUow{accRepo: accRepo, txRepo: txRepo}
	bus := eventbus.NewSimpleEventBus()

	// Prepare account read DTO
	accRead := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  amount.Amount(),
		Currency: amount.Currency().String(),
	}

	// Expectations
	accRepo.On("Get", mock.Anything, accountID).Return(accRead, nil)
	txRepo.On("Create", mock.Anything, mock.AnythingOfType("dto.TransactionCreate")).Return(nil)

	// Track event emissions
	emitted := make([]string, 0, 10)
	var mu sync.Mutex
	track := func(eventType string) {
		mu.Lock()
		emitted = append(emitted, eventType)
		mu.Unlock()
	}

	// Register handlers (real logic)
	bus.Subscribe("DepositRequestedEvent", func(ctx context.Context, e domain.Event) {
		track("DepositRequestedEvent")
		deposithandler.ValidationHandler(bus, uow, nil)(ctx, e)
	})
	bus.Subscribe("DepositValidatedEvent", func(ctx context.Context, e domain.Event) {
		track("DepositValidatedEvent")
		deposithandler.PersistenceHandler(bus, uow, nil)(ctx, e)
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

	accRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}