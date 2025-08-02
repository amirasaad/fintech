package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/commands"
	"github.com/amirasaad/fintech/pkg/domain/events"

	"log/slog"

	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
	gotAccount, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
	require.Error(t, err)
	assert.Empty(t, gotAccount)
}

func TestDeposit_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewWithMemory(slog.Default())
	svc := accountsvc.NewService(memBus, nil, slog.Default())

	var called bool

	userID := uuid.New()
	accountID := uuid.New()
	amount := 100.0
	currencyCode := "USD"

	// Register the handler before publishing
	memBus.Register("DepositRequested", func(c context.Context, e events.Event) error {
		evt, ok := e.(*events.DepositRequested)
		require.True(t, ok)
		assert.Equal(t, userID, evt.UserID)
		assert.Equal(t, accountID, evt.AccountID)
		assert.InEpsilon(t, amount, evt.Amount.AmountFloat(), 0.01)
		assert.Equal(t, currencyCode, evt.Amount.Currency().String())
		called = true
		return nil
	})

	err := svc.Deposit(context.Background(), commands.Deposit{
		UserID:    userID,
		AccountID: accountID,
		Amount:    amount,
		Currency:  currencyCode,
	})
	require.NoError(t, err)
	assert.True(t, called, "Handler should have been called")
}

func TestWithdraw_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewWithMemory(slog.Default())
	svc := accountsvc.NewService(memBus, nil, slog.Default())
	userID := uuid.New()
	accountID := uuid.New()
	var publishedEvents []events.Event
	memBus.Register("WithdrawRequested", func(c context.Context, e events.Event) error {
		publishedEvents = append(publishedEvents, e)
		return nil
	})
	err := svc.Withdraw(context.Background(), commands.Withdraw{
		UserID:    userID,
		AccountID: accountID,
		Amount:    50.0,
		Currency:  "USD",
		ExternalTarget: &commands.ExternalTarget{
			BankAccountNumber: "1234567890",
		},
	})
	require.NoError(t, err)
	require.Len(t, publishedEvents, 1)
	evt, ok := publishedEvents[0].(*events.WithdrawRequested)
	require.True(t, ok)
	assert.Equal(t, userID, evt.UserID)
	assert.Equal(t, accountID, evt.AccountID)
	assert.InEpsilon(t, 50.0, evt.Amount.AmountFloat(), 0.01)
	assert.Equal(t, "USD", evt.Amount.Currency().String())
	assert.Equal(t, "1234567890", evt.BankAccountNumber)
}

func TestTransfer_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewWithMemory(slog.Default())
	userID := uuid.New()
	sourceAccountID := uuid.New()
	destAccountID := uuid.New()
	amount := 25.0
	currency := "USD"

	svc := accountsvc.NewService(memBus, nil, slog.Default())
	var publishedEvents []events.Event
	memBus.Register("TransferRequested", func(c context.Context, e events.Event) error {
		publishedEvents = append(publishedEvents, e)
		return nil
	})
	cmd := commands.Transfer{
		UserID:      userID,
		AccountID:   sourceAccountID,
		ToAccountID: destAccountID,
		Amount:      amount,
		Currency:    currency,
	}
	err := svc.Transfer(context.TODO(), cmd, destAccountID)
	require.NoError(t, err)
	require.Len(t, publishedEvents, 1)
	evt, ok := publishedEvents[0].(*events.TransferRequested)
	require.True(t, ok)
	assert.Equal(t, userID, evt.UserID)
	assert.Equal(t, sourceAccountID, evt.AccountID)
	assert.Equal(t, destAccountID, evt.DestAccountID)
	// assert.InEpsilon(t, amount, evt.Amount, 0.01)
	assert.Equal(t, currency, evt.Amount.Currency().String())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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

	svc := accountsvc.NewService(nil, uow, slog.Default())
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
	err := acc.ValidateDeposit(userID, balanceMoney)
	require.NoError(t, err)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	_, _ = accountsvc.NewService(nil, uow, slog.Default()).GetBalance(userID, acc.ID)

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

	accountRepo.EXPECT().Get(mock.Anything).Return(&accountdomain.Account{}, accountdomain.ErrAccountNotFound)

	balance, err := accountsvc.NewService(nil, uow, slog.Default()).GetBalance(uuid.New(), uuid.New())
	require.Error(err)
	assert.InDelta(0, balance, 0.01)
}
