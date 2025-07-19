package webapi_test

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
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	deposithandler "github.com/amirasaad/fintech/pkg/handler/account/deposit"
	"github.com/amirasaad/fintech/pkg/handler/payment"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"
	mocks "github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/mock"
)

type E2ETestSuite struct {
	suite.Suite
	bus     *eventbus.SimpleEventBus
	accRepo *mocks.AccountRepository
	txRepo  *mocks.TransactionRepository
	uow     *mockUow
	ctx     context.Context
	mu      sync.Mutex
	events  []string
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

func (s *E2ETestSuite) SetupTest() {
	s.ctx = context.Background()
	s.bus = eventbus.NewSimpleEventBus()
	s.accRepo = mocks.NewAccountRepository(s.T())
	s.txRepo = mocks.NewTransactionRepository(s.T())
	s.uow = &mockUow{accRepo: s.accRepo, txRepo: s.txRepo}
	s.events = nil
}

func (s *E2ETestSuite) track(eventType string) {
	s.mu.Lock()
	s.events = append(s.events, eventType)
	s.mu.Unlock()
}

func (s *E2ETestSuite) TestDepositE2EFlow() {
	userID := uuid.New()
	accountID := uuid.New()
	amount, _ := money.New(100, currency.USD)

	accRead := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  amount.Amount(),
		Currency: amount.Currency().String(),
	}

	s.accRepo.On("Get", s.ctx, accountID).Return(accRead, nil)
	s.txRepo.On("Create", s.ctx, mock.AnythingOfType("dto.TransactionCreate")).Return(nil)

	s.bus.Subscribe("DepositRequestedEvent", func(ctx context.Context, e domain.Event) {
		s.track("DepositRequestedEvent")
		deposithandler.ValidationHandler(s.bus, s.uow, nil)(ctx, e)
	})
	s.bus.Subscribe("DepositValidatedEvent", func(ctx context.Context, e domain.Event) {
		s.track("DepositValidatedEvent")
		deposithandler.PersistenceHandler(s.bus, s.uow, nil)(ctx, e)
	})
	s.bus.Subscribe("DepositPersistedEvent", func(ctx context.Context, e domain.Event) {
		s.track("DepositPersistedEvent")
	})

	s.bus.Publish(s.ctx, events.DepositRequestedEvent{
		EventID:   uuid.New(),
		AccountID: accountID,
		UserID:    userID,
		Amount:    amount,
		Source:    "deposit",
		Timestamp: time.Now(),
	})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(s.T(), []string{
		"DepositRequestedEvent",
		"DepositValidatedEvent",
		"DepositPersistedEvent",
	}, s.events, "event chain should match deposit flow")

	s.accRepo.AssertExpectations(s.T())
	s.txRepo.AssertExpectations(s.T())
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}