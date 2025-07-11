package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper to create a service with mocks
func newServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (svc *AccountService, accountRepo *fixtures.MockAccountRepository, transactionRepo *fixtures.MockTransactionRepository, uow *fixtures.MockUnitOfWork) {
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)
	uow = fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Maybe()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Maybe()
	svc = NewAccountService(uow, nil, slog.Default())
	return svc, accountRepo, transactionRepo, uow
}

func TestCreateAccount_Success(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)

	accountRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	userID := uuid.New()
	gotAccount, err := svc.CreateAccount(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotNil(t, gotAccount)
	assert.Equal(t, userID, gotAccount.UserID)
}

func TestCreateAccount_RepoError(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)

	accountRepo.EXPECT().Create(mock.Anything).Return(errors.New("db error")).Once()

	userID := uuid.New()
	gotAccount, err := svc.CreateAccount(context.Background(), userID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestDeposit_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).Once()

	userID := uuid.New()
	acc, err := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	require.NoError(t, err)
	m, err := money.NewMoney(100.0, currency.Code("USD"))
	require.NoError(t, err)
	_, _ = acc.Deposit(userID, m)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	gotTx, _, err := svc.Deposit(userID, acc.ID, 50.0, currency.Code("USD"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.Equal(t, acc.ID, gotTx.AccountID)
	assert.Equal(t, userID, gotTx.UserID)
	assert.Equal(t, currency.Code("USD"), gotTx.Currency)
}

func TestDeposit_RepoError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(3)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	accountRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("repo error")).Once()

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_NegativeAmount(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(3)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build()
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil).Once()
	gotTx, _, err := svc.Deposit(userID, acc.ID, -50.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
	assert.Equal(t, domain.ErrTransactionAmountMustBePositive, err)
}

func TestDeposit_AccountRepoError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(3)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	accountRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("account repo error")).Once()

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_GetAccountError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil)
	accountRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("get error"))

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
	assert.ErrorIs(t, err, domain.ErrAccountNotFound)
}

func TestDeposit_UpdateError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil)
	acc, _ := account.New().WithUserID(uuid.New()).WithCurrency(currency.USD).Build()
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(mock.Anything).Return(errors.New("update error"))

	gotTx, _, err := svc.Deposit(acc.UserID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_TransactionRepoError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil)
	acc, _ := account.New().WithUserID(uuid.New()).WithCurrency(currency.USD).Build()
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(mock.Anything).Return(nil)
	uow.EXPECT().GetRepository(mock.Anything).Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(errors.New("create error"))

	gotTx, _, err := svc.Deposit(acc.UserID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_CommitError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(3)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(errors.New("commit error")).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return errors.New("commit error")
		},
	).Once()

	acc, _ := account.New().WithUserID(uuid.New()).WithCurrency(currency.USD).Build()
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	gotTx, _, err := svc.Deposit(acc.UserID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestWithdraw_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build()
	// Deposit first
	amount, _ := money.NewMoney(100, currency.Code("USD"))
	_, _ = acc.Deposit(userID, amount)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(acc).Return(nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	gotTx, _, err := svc.Withdraw(userID, acc.ID, 50.0, currency.Code("USD"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	balance, _ := acc.GetBalance(userID)
	assert.InDelta(t, 50.0, balance, 0.01)
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &account.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(1)
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Withdraw(userID, accountID, 50.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestWithdraw_UoWFactoryError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := NewAccountService(uow, nil, slog.Default())
	_, _, err := svc.Withdraw(uuid.New(), uuid.New(), 100.0, currency.USD)
	assert.Error(t, err)
}

func TestWithdraw_UpdateError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &account.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 100}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(2)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(1)
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(errors.New("update error")).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Withdraw(userID, accountID, 50.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestGetAccount_Success(t *testing.T) {
	t.Parallel()
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &account.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 100}

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.GetAccount(userID, accountID)
	assert.NoError(t, err)
	assert.NotNil(t, gotAccount)
}

func TestGetAccount_NotFound(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(nil, account.ErrAccountNotFound).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.GetAccount(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetAccount_UoWFactoryError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := NewAccountService(uow, nil, slog.Default())
	_, err := svc.GetAccount(uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetAccount_Unauthorized(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(nil, user.ErrUserUnauthorized).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.GetAccount(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetTransactions_Success(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	txs := []*account.Transaction{
		{ID: uuid.New(), UserID: userID, AccountID: accountID, Amount: 100, Currency: currency.USD},
	}

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	transactionRepo.EXPECT().List(userID, accountID).Return(txs, nil).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	gotTxs, err := svc.GetTransactions(userID, accountID)
	assert.NoError(t, err)
	assert.Equal(t, txs, gotTxs)
}

func TestGetTransactions_Error(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	transactionRepo.EXPECT().List(userID, accountID).Return(nil, errors.New("list error")).Once()

	svc := NewAccountService(uow, nil, slog.Default())
	txs, err := svc.GetTransactions(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, txs)
}

func TestGetTransactions_UoWFactoryError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := NewAccountService(uow, nil, slog.Default())
	_, err := svc.GetTransactions(uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build()
	balanceMoney, err := money.NewMoney(123.45, currency.Code("USD"))
	require.NoError(t, err)
	_, _ = acc.Deposit(userID, balanceMoney)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)

	balance, err := svc.GetBalance(userID, acc.ID)
	assert.NoError(t, err)
	assert.InDelta(t, 123.45, balance, 0.01)
}

func TestGetBalance_NotFound(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	balance, err := svc.GetBalance(uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, 0.0, balance)
}

func TestGetBalance_UoWFactoryError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := NewAccountService(uow, nil, slog.Default())
	_, err := svc.GetBalance(uuid.New(), uuid.New())
	assert.Error(t, err)
}
