package service

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper to create a service with mocks
func newServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (scv *AccountService, accountRepo *test.MockAccountRepository, transactionRepo *test.MockTransactionRepository, uow *test.MockUnitOfWork) {
	accountRepo = test.NewMockAccountRepository(t)
	transactionRepo = test.NewMockTransactionRepository(t)
	uow = test.NewMockUnitOfWork(t)
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil })
	return svc, accountRepo, transactionRepo, uow
}

func TestCreateAccount_Success(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.On("Create", mock.Anything).Return(nil)

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
	uow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.On("Create", mock.Anything).Return(errors.New("db error"))

	userID := uuid.New()
	account, err := svc.CreateAccount(userID)
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestDeposit_Success(t *testing.T) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo)
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Once()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.On("Get", account.ID).Return(account, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)

	tx, err := svc.Deposit(userID, account.ID, 100.0)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	balance, _ := account.GetBalance(userID)
	assert.InDelta(t, 100.0, balance, 0.01)
}

func TestDeposit_AccountNotFound(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	tx, err := svc.Deposit(uuid.New(), uuid.New(), 100.0)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestDeposit_NegativeAmount(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.On("Get", account.ID).Return(account, nil)

	tx, err := svc.Deposit(userID, account.ID, -50.0)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrTransactionAmountMustBePositive, err)
}

func TestWithdraw_Success(t *testing.T) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().AccountRepository().Return(accountRepo)
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Once()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	// Deposit first
	_, _ = account.Deposit(userID, 100.0)
	accountRepo.On("Get", account.ID).Return(account, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)

	tx, err := svc.Withdraw(userID, account.ID, 50.0)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	balance, _ := account.GetBalance(userID)
	assert.InDelta(t, 50.0, balance, 0.01)
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.On("Get", account.ID).Return(account, nil)

	tx, err := svc.Withdraw(userID, account.ID, 100.0)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrInsufficientFunds, err)
}

func TestGetAccount_Success(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo)
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.On("Get", account.ID).Return(account, nil)

	got, err := svc.GetAccount(userID, account.ID)
	assert.NoError(t, err)
	assert.Equal(t, account, got)
}

func TestGetAccount_NotFound(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	got, err := svc.GetAccount(uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestGetTransactions_Success(t *testing.T) {
	svc, _, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Once()

	accountID := uuid.New()
	userID := uuid.New()
	txList := []*domain.Transaction{
		{ID: uuid.New(), AccountID: accountID, Amount: 100, Balance: 100},
	}
	transactionRepo.On("List", userID, accountID).Return(txList, nil)

	got, err := svc.GetTransactions(userID, accountID)
	assert.NoError(t, err)
	assert.Equal(t, txList, got)
}

func TestGetTransactions_Error(t *testing.T) {
	svc, _, transactionRepo, uow := newServiceWithMocks(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Once()

	accountID := uuid.New()
	userID := uuid.New()
	transactionRepo.On("List", userID, accountID).Return([]*domain.Transaction{}, errors.New("db error"))

	got, err := svc.GetTransactions(userID, accountID)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetBalance_Success(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo)

	userID := uuid.New()
	account := domain.NewAccount(userID)
	_, _ = account.Deposit(userID, 123.45)
	accountRepo.On("Get", account.ID).Return(account, nil)

	balance, err := svc.GetBalance(userID, account.ID)
	assert.NoError(t, err)
	assert.InDelta(t, 123.45, balance, 0.01)
}

func TestGetBalance_NotFound(t *testing.T) {
	svc, accountRepo, _, uow := newServiceWithMocks(t)
	uow.EXPECT().AccountRepository().Return(accountRepo)

	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	balance, err := svc.GetBalance(uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, 0.0, balance)
}

func BenchmarkCreateAccount(b *testing.B) {
	svc, accountRepo, _, uow := newServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	uow.EXPECT().AccountRepository().Return(accountRepo).Maybe()
	accountRepo.On("Create", mock.Anything).Return(nil).Maybe()
	userID := uuid.New()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.CreateAccount(userID)
	}
}

func BenchmarkDeposit(b *testing.B) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	uow.EXPECT().AccountRepository().Return(accountRepo).Maybe()
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Maybe()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.On("Get", account.ID).Return(account, nil).Maybe()
	accountRepo.On("Update", mock.Anything).Return(nil).Maybe()
	transactionRepo.On("Create", mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.Deposit(userID, account.ID, 100.0)
	}
}

func BenchmarkWithdraw(b *testing.B) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	uow.EXPECT().AccountRepository().Return(accountRepo).Maybe()
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Maybe()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	_, _ = account.Deposit(userID, float64(50*b.N))
	accountRepo.On("Get", account.ID).Return(account, nil).Maybe()
	accountRepo.On("Update", mock.Anything).Return(nil).Maybe()
	transactionRepo.On("Create", mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.Withdraw(userID, account.ID, 50.0)
	}
}
