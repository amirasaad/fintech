package account_test

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/commands"
	"github.com/amirasaad/fintech/pkg/domain/events"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	"github.com/amirasaad/fintech/pkg/repository/transaction"

	"log/slog"

	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestMocks(t *testing.T) (
	*mocks.UnitOfWork,
	*mocks.AccountRepository,
	*mocks.TransactionRepository,
) {
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	transactionRepo := mocks.NewTransactionRepository(t)
	return uow, accountRepo, transactionRepo
}

func TestCreateAccount_Success(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
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

	svc := accountsvc.New(nil, uow, slog.Default())
	_, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
	require.NoError(t, err)
}

func TestCreateAccount_RepoError(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	userID := uuid.New()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil).Once()
	accountRepo.EXPECT().Create(
		mock.Anything,
		mock.Anything,
	).Return(errors.New("repo error")).Once()

	svc := accountsvc.New(nil, uow, slog.Default())
	gotAccount, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
	require.Error(t, err)
	assert.Empty(t, gotAccount)
}

func TestDeposit_PublishesEvent(t *testing.T) {
	memBus := eventbus.NewWithMemory(slog.Default())
	svc := accountsvc.New(memBus, nil, slog.Default())

	var called bool

	userID := uuid.New()
	accountID := uuid.New()
	amount := 100.0
	currencyCode := "USD"

	// Register the handler before publishing
	memBus.Register(
		events.EventTypeDepositRequested,
		func(c context.Context, e events.Event) error {
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
	svc := accountsvc.New(memBus, nil, slog.Default())
	userID := uuid.New()
	accountID := uuid.New()
	var publishedEvents []events.Event
	memBus.Register(
		events.EventTypeWithdrawRequested,
		func(c context.Context, e events.Event) error {
			publishedEvents = append(publishedEvents, e)
			return nil
		},
	)
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

	svc := accountsvc.New(memBus, nil, slog.Default())
	var publishedEvents []events.Event
	memBus.Register(
		events.EventTypeTransferRequested,
		func(c context.Context, e events.Event) error {
			publishedEvents = append(publishedEvents, e)
			return nil
		},
	)
	cmd := commands.Transfer{
		UserID:      userID,
		AccountID:   sourceAccountID,
		ToAccountID: destAccountID,
		Amount:      amount,
		Currency:    currency,
	}
	err := svc.Transfer(context.TODO(), cmd)
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
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  100,
		Currency: "USD",
	}

	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil).Once()
	accountRepo.EXPECT().Get(context.Background(), accountID).Return(account, nil).Once()

	svc := accountsvc.New(nil, uow, slog.Default())
	gotAccount, err := svc.GetAccount(context.Background(), userID, accountID)
	require.NoError(t, err)
	assert.NotNil(t, gotAccount)
}

func TestGetAccount_NotFound(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil).Once()

	accountRepo.EXPECT().Get(context.Background(), accountID).
		Return(nil, accountdomain.ErrAccountNotFound).Once()

	svc := accountsvc.New(nil, uow, slog.Default())
	gotAccount, err := svc.GetAccount(context.Background(), userID, accountID)
	require.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetAccount_Unauthorized(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil).Times(1)

	accountRepo.EXPECT().Get(context.Background(), accountID).
		Return(nil, user.ErrUserUnauthorized).Once()

	svc := accountsvc.New(nil, uow, slog.Default())
	gotAccount, err := svc.GetAccount(context.Background(), userID, accountID)
	require.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetTransactions_Success(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	transactionRepo := mocks.NewTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  100,
		Currency: "USD",
	}
	txs := []*dto.TransactionRead{
		{ID: uuid.New(), UserID: userID, AccountID: accountID,
			Amount:   100,
			Currency: "USD",
		},
	}

	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository((*transaction.Repository)(nil)).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(context.Background(), accountID).Return(account, nil).Once()
	transactionRepo.EXPECT().ListByAccount(context.Background(), accountID).Return(txs, nil).Once()

	svc := accountsvc.New(nil, uow, slog.Default())
	gotTxs, err := svc.GetTransactions(context.Background(), userID, accountID)
	require.NoError(t, err)
	assert.Equal(t, txs, gotTxs)
}

func TestGetTransactions_Error(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
	accountRepo := mocks.NewAccountRepository(t)
	transactionRepo := mocks.NewTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &dto.AccountRead{
		ID:       accountID,
		UserID:   userID,
		Balance:  100,
		Currency: "USD",
	}

	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository((*transaction.Repository)(nil)).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(context.Background(), accountID).Return(account, nil).Once()
	transactionRepo.EXPECT().ListByAccount(
		context.Background(),
		accountID,
	).Return(nil, errors.New("list error")).Once()

	svc := accountsvc.New(nil, uow, slog.Default())
	txs, err := svc.GetTransactions(context.Background(), userID, accountID)
	require.Error(t, err)
	assert.Nil(t, txs)
}

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()
	uow, accountRepo, _ := setupTestMocks(t)
	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil)

	userID := uuid.New()
	acc := dto.AccountRead{
		ID:       uuid.New(),
		UserID:   userID,
		Balance:  100,
		Currency: "USD",
	}
	accountRepo.EXPECT().Get(context.Background(), acc.ID).Return(&acc, nil)
	svc := accountsvc.New(nil, uow, slog.Default())
	got, err := svc.GetBalance(context.Background(), userID, acc.ID)
	require.NoError(t, err)
	assert.InDelta(t, acc.Balance, got, 0.01)

}

func TestGetBalance_NotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	uow, accountRepo, _ := setupTestMocks(t)
	uow.EXPECT().GetRepository((*repoaccount.Repository)(nil)).Return(accountRepo, nil)

	accountRepo.EXPECT().Get(
		mock.Anything,
		mock.Anything,
	).Return(
		nil,
		accountdomain.ErrAccountNotFound,
	)

	balance, err := accountsvc.New(
		nil,
		uow,
		slog.Default(),
	).GetBalance(context.Background(), uuid.New(), uuid.New())
	require.Error(err)
	assert.InDelta(0, balance, 0.01)
}
