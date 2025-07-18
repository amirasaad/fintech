package account_test

import (
	"context"
	"errors"
	"testing"

	events2 "github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper to create a service with mocks
func newServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (svc *accountsvc.Service, accountRepo *mocks.MockAccountRepository, transactionRepo *mocks.MockTransactionRepository, uow *mocks.MockUnitOfWork) {
	accountRepo = mocks.NewMockAccountRepository(t)
	transactionRepo = mocks.NewMockTransactionRepository(t)
	uow = mocks.NewMockUnitOfWork(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Maybe()
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Maybe()
	svc = accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	return svc, accountRepo, transactionRepo, uow
}

func setupTestMocks(t *testing.T) (*mocks.MockUnitOfWork, *mocks.MockAccountRepository, *mocks.MockTransactionRepository) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	transactionRepo := mocks.NewMockTransactionRepository(t)
	return uow, accountRepo, transactionRepo
}

func TestCreateAccount_Success(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	userID := uuid.New()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil).Once()
	accountRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	accountRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(&dto.AccountRead{}, nil).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	_, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
	require.NoError(t, err)
}

func TestCreateAccount_RepoError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	userID := uuid.New()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil).Once()
	accountRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("repo error")).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	gotAccount, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
	require.Error(t, err)
	assert.Empty(t, gotAccount)
}

func TestDeposit_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewMemoryEventBus()
	svc := accountsvc.NewService(config.Deps{
		Uow:               nil,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
		EventBus:          memBus,
	})

	// Register the handler before publishing
	var called bool
	memBus.Subscribe("DepositRequestedEvent", func(c context.Context, e domain.Event) {
		called = true
		// Optionally, assert on event fields here
	})

	err := svc.Deposit(context.Background(), dto.TransactionCommand{Amount: 100})
	require.NoError(t, err)
	assert.True(t, called, "Handler should have been called")
}

func TestWithdraw_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewMemoryEventBus()
	svc := accountsvc.NewService(config.Deps{
		Uow:               nil,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
		EventBus:          memBus,
	})
	userID := uuid.New()
	accountID := uuid.New()
	var publishedEvents []domain.Event
	memBus.Subscribe("WithdrawRequestedEvent", func(c context.Context, e domain.Event) {
		publishedEvents = append(publishedEvents, e)
	})
	externalTarget := accountdomain.ExternalTarget{BankAccountNumber: "1234567890"}
	err := svc.Withdraw(userID, accountID, 50.0, currency.USD, externalTarget)
	require.NoError(t, err)
	require.Len(t, publishedEvents, 1)
	evt, ok := publishedEvents[0].(events2.WithdrawRequestedEvent)
	require.True(t, ok)
	assert.Equal(t, userID, evt.UserID)
	assert.Equal(t, accountID, evt.AccountID)
	assert.InEpsilon(t, 50.0, evt.Amount.AmountFloat(), 0.01)
	assert.Equal(t, "USD", evt.Amount.Currency().String())
	assert.Equal(t, externalTarget.BankAccountNumber, evt.BankAccountNumber)
}

func TestTransfer_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewMemoryEventBus()
	userID := uuid.New()
	sourceAccountID := uuid.New()
	destAccountID := uuid.New()
	amount := 25.0
	currency := "USD"
	moneySource := string(accountdomain.MoneySourceInternal)

	svc := accountsvc.NewService(config.Deps{
		EventBus: memBus,
	})
	var publishedEvents []domain.Event
	memBus.Subscribe("TransferRequestedEvent", func(c context.Context, e domain.Event) {
		publishedEvents = append(publishedEvents, e)
	})
	transferDTO := dto.TransactionCreate{
		UserID:      userID,
		AccountID:   sourceAccountID,
		Amount:      int64(amount), // TODO: until refactor
		Currency:    currency,
		MoneySource: moneySource,
	}
	err := svc.Transfer(context.TODO(), transferDTO, destAccountID)
	require.NoError(t, err)
	require.Len(t, publishedEvents, 1)
	evt, ok := publishedEvents[0].(events2.TransferRequestedEvent)
	require.True(t, ok)
	assert.Equal(t, userID, evt.SenderUserID)
	assert.Equal(t, sourceAccountID, evt.SourceAccountID)
	assert.Equal(t, destAccountID, evt.DestAccountID)
	// assert.InEpsilon(t, amount, evt.Amount, 0.01)
	assert.Equal(t, currency, evt.Amount.Currency().String())
	assert.Equal(t, moneySource, evt.Source)
}

func TestGetAccount_Success(t *testing.T) {
	t.Parallel()
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Balance: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(100, currency.USD); return m }()}

	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	gotAccount, err := svc.GetAccount(userID, accountID)
	require.NoError(t, err)
	assert.NotNil(t, gotAccount)
}

func TestGetAccount_NotFound(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(nil, accountdomain.ErrAccountNotFound).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	gotAccount, err := svc.GetAccount(userID, accountID)
	require.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetAccount_Unauthorized(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(nil, user.ErrUserUnauthorized).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	gotAccount, err := svc.GetAccount(userID, accountID)
	require.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetTransactions_Success(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	transactionRepo := mocks.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Balance: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(0, currency.USD); return m }()}
	txs := []*accountdomain.Transaction{
		{ID: uuid.New(), UserID: userID, AccountID: accountID, Amount: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(100, currency.USD); return m }()},
	}

	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	transactionRepo.EXPECT().List(userID, accountID).Return(txs, nil).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	gotTxs, err := svc.GetTransactions(userID, accountID)
	require.NoError(t, err)
	assert.Equal(t, txs, gotTxs)
}

func TestGetTransactions_Error(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	transactionRepo := mocks.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Balance: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(0, currency.USD); return m }()}

	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	transactionRepo.EXPECT().List(userID, accountID).Return(nil, errors.New("list error")).Once()

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	txs, err := svc.GetTransactions(userID, accountID)
	require.Error(t, err)
	assert.Nil(t, txs)
}

func TestGetTransactions_UoWFactoryError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	_, err := svc.GetTransactions(uuid.New(), uuid.New())
	require.Error(t, err)
}

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()
	uow, accountRepo, _ := setupTestMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)

	userID := uuid.New()
	acc, _ := accountdomain.New().WithUserID(userID).WithCurrency(currency.USD).Build()
	balanceMoney, _ := money.New(123.0, acc.Balance.Currency())
	_ = acc.Deposit(userID, balanceMoney, accountdomain.MoneySourceCard, "")
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	_, _ = accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	}).GetBalance(userID, acc.ID)

}

func TestGetBalance_NotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	uow, accountRepo, _ := setupTestMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	balance, err := accountsvc.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: nil,
		Logger:            slog.Default(),
		PaymentProvider:   provider.NewMockPaymentProvider(),
	}).GetBalance(uuid.New(), uuid.New())
	require.Error(err)
	assert.InDelta(0, balance, 0.01)
}
