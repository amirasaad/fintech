package service

import (
	"errors"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
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
}) (scv *AccountService, accountRepo *fixtures.MockAccountRepository, transactionRepo *fixtures.MockTransactionRepository, uow *fixtures.MockUnitOfWork) {
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)
	uow = fixtures.NewMockUnitOfWork(t)
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, nil, slog.Default())
	return svc, accountRepo, transactionRepo, uow
}

func TestCreateAccount_Success(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Create(mock.Anything).Return(nil)

	userID := uuid.New()
	gotAccount, err := svc.CreateAccount(userID)
	assert.NoError(t, err)
	assert.NotNil(t, gotAccount)
	assert.Equal(t, userID, gotAccount.UserID)
}

func TestCreateAccount_RepoError(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))

	userID := uuid.New()
	gotAccount, err := svc.CreateAccount(userID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestCreateAccount_UoWFactoryError(t *testing.T) {
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil, slog.Default())
	gotAccount, err := svc.CreateAccount(uuid.New())
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestCreateAccount_CommitError(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(errors.New("commit error")).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Create(mock.Anything).Return(nil)

	userID := uuid.New()
	gotAccount, err := svc.CreateAccount(userID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
}

func TestDeposit_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()
	userID := uuid.New()
	// Create an account and deposit some funds
	acc := account.NewAccount(userID)
	m, err := money.NewMoney(100.0, currency.Code("USD"))
	require.NoError(t, err)
	_, _ = acc.Deposit(userID, m)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(mock.Anything).Return(nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Test deposit
	gotTx, _, err := svc.Deposit(userID, acc.ID, 50.0, currency.Code("USD"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.Equal(t, acc.ID, gotTx.AccountID)
	assert.Equal(t, userID, gotTx.UserID)
	assert.Equal(t, currency.Code("USD"), gotTx.Currency)
}

func TestDeposit_AccountNotFound(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Get(mock.Anything).Return(nil, domain.ErrAccountNotFound)

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
	assert.ErrorIs(t, err, domain.ErrAccountNotFound)
}

func TestDeposit_NegativeAmount(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	acc := account.NewAccount(userID)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	gotTx, _, err := svc.Deposit(userID, acc.ID, -50.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
	assert.Equal(t, domain.ErrTransactionAmountMustBePositive, err)
}

func TestDeposit_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil, slog.Default())
	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_BeginError(t *testing.T) {
	t.Parallel()
	svc, _, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(errors.New("begin error"))

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_AccountRepoError(t *testing.T) {
	t.Parallel()
	svc, _, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().AccountRepository().Return(nil, errors.New("repo error"))
	uow.EXPECT().Rollback().Return(nil)

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_GetAccountError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Get(mock.Anything).Return(nil, errors.New("get error"))
	uow.EXPECT().Rollback().Return(nil)

	gotTx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
	assert.ErrorIs(t, err, domain.ErrAccountNotFound)
}

func TestDeposit_UpdateError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	acc := account.NewAccount(uuid.New())
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(mock.Anything).Return(errors.New("update error"))
	uow.EXPECT().Rollback().Return(nil)

	gotTx, _, err := svc.Deposit(acc.UserID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_TransactionRepoError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	acc := account.NewAccount(uuid.New())
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(mock.Anything).Return(nil)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(errors.New("create error"))
	uow.EXPECT().Rollback().Return(nil)

	gotTx, _, err := svc.Deposit(acc.UserID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestDeposit_CommitError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	acc := account.NewAccount(uuid.New())
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(mock.Anything).Return(nil)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)
	uow.EXPECT().Commit().Return(errors.New("commit error"))
	uow.EXPECT().Rollback().Return(nil)

	gotTx, _, err := svc.Deposit(acc.UserID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestWithdraw_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()
	userID := uuid.New()
	acc := account.NewAccount(userID)
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
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	userID := uuid.New()
	acc := account.NewAccount(userID)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)

	gotTx, _, err := svc.Withdraw(userID, acc.ID, 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
	assert.Equal(t, domain.ErrInsufficientFunds, err)
}

func TestWithdraw_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil, slog.Default())
	gotTx, _, err := svc.Withdraw(uuid.New(), uuid.New(), 100.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestWithdraw_UpdateError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	acc := account.NewAccount(userID)
	// Deposit first
	amount, _ := money.NewMoney(100, currency.Code("USD"))
	_, _ = acc.Deposit(userID, amount)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)
	accountRepo.EXPECT().Update(acc).Return(errors.New("update error"))

	gotTx, _, err := svc.Withdraw(userID, acc.ID, 50.0, currency.Code("USD"))
	assert.Error(t, err)
	assert.Nil(t, gotTx)
}

func TestGetAccount_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	acc := account.NewAccount(userID)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)

	gotAccount, err := svc.GetAccount(userID, acc.ID)
	assert.NoError(t, err)
	assert.Equal(t, acc, gotAccount)
}

func TestGetAccount_NotFound(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	gotAccount, err := svc.GetAccount(uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestGetAccount_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil, slog.Default())
	_, err := svc.GetAccount(uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetAccount_Unauthorized(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	acc := account.NewAccount(userID)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil)

	gotAccount, err := svc.GetAccount(uuid.New(), acc.ID)
	assert.Error(t, err)
	assert.Nil(t, gotAccount)
	assert.Equal(t, domain.ErrUserUnauthorized, err)
}

func TestGetTransactions_Success(t *testing.T) {
	t.Parallel()
	svc, _, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()

	accountID := uuid.New()
	userID := uuid.New()
	txList := []*domain.Transaction{
		{ID: uuid.New(), AccountID: accountID, Amount: 100, Balance: 100},
	}
	transactionRepo.EXPECT().List(userID, accountID).Return(txList, nil)

	got, err := svc.GetTransactions(userID, accountID)
	assert.NoError(t, err)
	assert.Equal(t, txList, got)
}

func TestGetTransactions_Error(t *testing.T) {
	t.Parallel()
	svc, _, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()

	accountID := uuid.New()
	userID := uuid.New()
	transactionRepo.EXPECT().List(userID, accountID).Return([]*domain.Transaction{}, errors.New("db error"))

	got, err := svc.GetTransactions(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetTransactions_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil, slog.Default())
	_, err := svc.GetTransactions(uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)

	userID := uuid.New()
	acc := account.NewAccount(userID)
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
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)

	accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	balance, err := svc.GetBalance(uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, 0.0, balance)
}

func TestGetBalance_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil, slog.Default())
	_, err := svc.GetBalance(uuid.New(), uuid.New())
	assert.Error(t, err)
}
