package service

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper to create a service with mocks
func newServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (scv *AccountService, accountRepo *fixtures.MockAccountRepository, transactionRepo *fixtures.MockTransactionRepository, uow *fixtures.MockUnitOfWork) {
	accountRepo = fixtures.NewMockAccountRepository(t)
	transactionRepo = fixtures.NewMockTransactionRepository(t)
	uow = fixtures.NewMockUnitOfWork(t)
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, nil)
	return svc, accountRepo, transactionRepo, uow
}

func TestCreateAccount_Success(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Create(mock.Anything).Return(nil)

	userID := uuid.New()
	account, err := svc.CreateAccount(userID)
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, userID, account.UserID)
}

func TestCreateAccount_RepoError(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))

	userID := uuid.New()
	account, err := svc.CreateAccount(userID)
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestCreateAccount_UoWFactoryError(t *testing.T) {
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil)
	account, err := svc.CreateAccount(uuid.New())
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestCreateAccount_CommitError(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(errors.New("commit error")).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Create(mock.Anything).Return(nil)

	userID := uuid.New()
	account, err := svc.CreateAccount(userID)
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestDeposit_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)
	accountRepo.EXPECT().Update(account).Return(nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	tx, _, err := svc.Deposit(userID, account.ID, 100.0, "USD")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	balance, _ := account.GetBalance(userID)
	assert.InDelta(t, 100.0, balance, 0.01)
}

func TestDeposit_AccountNotFound(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Get(mock.Anything).Return(nil, domain.ErrAccountNotFound)

	tx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestDeposit_NegativeAmount(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)
	tx, _, err := svc.Deposit(userID, account.ID, -50.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrTransactionAmountMustBePositive, err)
}

func TestDeposit_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil)
	tx, _, err := svc.Deposit(uuid.New(), uuid.New(), 100.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestDeposit_UpdateError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)
	accountRepo.EXPECT().Update(account).Return(errors.New("update error"))

	tx, _, err := svc.Deposit(userID, account.ID, 100.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestWithdraw_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Deposit first
	_, _ = account.Deposit(userID, domain.Money{Amount: 100.0, Currency: "USD"})
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)
	accountRepo.EXPECT().Update(account).Return(nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	tx, _, err := svc.Withdraw(userID, account.ID, 50.0, "USD")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	balance, _ := account.GetBalance(userID)
	assert.InDelta(t, 50.0, balance, 0.01)
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)

	tx, _, err := svc.Withdraw(userID, account.ID, 100.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrInsufficientFunds, err)
}

func TestWithdraw_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil)
	tx, _, err := svc.Withdraw(uuid.New(), uuid.New(), 100.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestWithdraw_UpdateError(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	_, _ = account.Deposit(userID, domain.Money{Amount: 100.0, Currency: "USD"})
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)
	accountRepo.EXPECT().Update(account).Return(errors.New("update error"))

	tx, _, err := svc.Withdraw(userID, account.ID, 50.0, "USD")
	assert.Error(t, err)
	assert.Nil(t, tx)
}

func TestGetAccount_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)

	got, err := svc.GetAccount(userID, account.ID)
	assert.NoError(t, err)
	assert.Equal(t, account, got)
}

func TestGetAccount_NotFound(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	accountRepo.EXPECT().Get(mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	got, err := svc.GetAccount(uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestGetAccount_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil)
	_, err := svc.GetAccount(uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetAccount_Unauthorized(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)

	got, err := svc.GetAccount(uuid.New(), account.ID)
	assert.Error(t, err)
	assert.Nil(t, got)
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
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil)
	_, err := svc.GetTransactions(uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo, nil)

	userID := uuid.New()
	account := domain.NewAccount(userID)
	_, _ = account.Deposit(userID, domain.Money{Amount: 123.45, Currency: "USD"})
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)

	balance, err := svc.GetBalance(userID, account.ID)
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
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") }, nil)
	_, err := svc.GetBalance(uuid.New(), uuid.New())
	assert.Error(t, err)
}
