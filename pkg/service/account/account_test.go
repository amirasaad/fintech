package account_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/domain/user"
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
}) (svc *accountsvc.AccountService, accountRepo *fixtures.MockAccountRepository, transactionRepo *fixtures.MockTransactionRepository, uow *fixtures.MockUnitOfWork) {
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)
	uow = fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Maybe()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Maybe()
	svc = accountsvc.NewAccountService(uow, nil, slog.Default())
	return svc, accountRepo, transactionRepo, uow
}

func TestCreateAccount_Success(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	userID := uuid.New()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	accountRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.CreateAccount(context.Background(), userID)
	assert.NoError(t, err)
	assert.NotNil(t, gotAccount)
	assert.Equal(t, userID, gotAccount.UserID)
}

func TestCreateAccount_RepoError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	userID := uuid.New()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	accountRepo.EXPECT().Create(mock.Anything).Return(errors.New("repo error")).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.CreateAccount(context.Background(), userID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestDeposit_Success(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(account).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, 100.0, currency.USD)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
}

func TestDeposit_RepoError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(account).Return(errors.New("update error")).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, 100.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestDeposit_NegativeAmount(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	// No update or create expected for negative amount

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, -50.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestDeposit_AccountRepoError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(nil, errors.New("get error")).Once()
	// No update or create expected if get fails

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, 100.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestDeposit_GetAccountError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(nil, accountdomain.ErrAccountNotFound).Once()
	// No update or create expected if get fails

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, 100.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestDeposit_UpdateError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(account).Return(errors.New("update error")).Once()
	// No create expected if update fails

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, 100.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestDeposit_TransactionRepoError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(account).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(errors.New("create error")).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Deposit(userID, accountID, 100.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
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
	acc, _ := accountdomain.New().WithUserID(userID).WithCurrency(currency.USD).Build()
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
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 10} // Not enough for withdrawal

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	tx, _, err := svc.Withdraw(userID, accountID, 50.0, currency.USD)
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestWithdraw_UpdateError(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 10000}

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(errors.New("update error")).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
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
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 100}

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
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
	accountRepo.EXPECT().Get(accountID).Return(nil, accountdomain.ErrAccountNotFound).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.GetAccount(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
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

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotAccount, err := svc.GetAccount(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestGetTransactions_Success(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}
	txs := []*accountdomain.Transaction{
		{ID: uuid.New(), UserID: userID, AccountID: accountID, Amount: 100, Currency: currency.USD},
	}

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Maybe()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	transactionRepo.EXPECT().List(userID, accountID).Return(txs, nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotTxs, err := svc.GetTransactions(userID, accountID)
	assert.NoError(t, err)
	assert.Equal(t, txs, gotTxs)
}

func TestGetTransactions_Error(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	userID := uuid.New()
	accountID := uuid.New()
	account := &accountdomain.Account{ID: accountID, UserID: userID, Currency: currency.USD, Balance: 0}

	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Maybe()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Times(1)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	transactionRepo.EXPECT().List(userID, accountID).Return(nil, errors.New("list error")).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
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

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
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
	acc, _ := accountdomain.New().WithUserID(userID).WithCurrency(currency.USD).Build()
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
