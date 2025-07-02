package service

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for repository.UnitOfWork and repositories

type MockAccountRepo struct {
	mock.Mock
	account *domain.Account
}

func (m *MockAccountRepo) Create(a *domain.Account) error {
	m.account = a
	args := m.Called(a)
	return args.Error(0)
}
func (m *MockAccountRepo) Get(id uuid.UUID) (*domain.Account, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *MockAccountRepo) Update(a *domain.Account) error {
	m.account = a
	args := m.Called(a)
	return args.Error(0)
}
func (m *MockAccountRepo) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

type MockTransactionRepo struct {
	mock.Mock
}

// Get implements repository.TransactionRepository.
func (m *MockTransactionRepo) Get(id uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepo) Create(tx *domain.Transaction) error {
	args := m.Called(tx)
	return args.Error(0)
}
func (m *MockTransactionRepo) List(accountID uuid.UUID) ([]*domain.Transaction, error) {
	args := m.Called(accountID)
	return args.Get(0).([]*domain.Transaction), args.Error(1)
}

type MockUoW struct {
	mock.Mock
	userRepo        *MockUserRepo // Not used in these tests
	accountRepo     *MockAccountRepo
	transactionRepo *MockTransactionRepo
}

func (m *MockUoW) Begin() error    { return nil }
func (m *MockUoW) Commit() error   { return nil }
func (m *MockUoW) Rollback() error { return nil }
func (m *MockUoW) AccountRepository() repository.AccountRepository {
	return m.accountRepo
}
func (m *MockUoW) TransactionRepository() repository.TransactionRepository {
	return m.transactionRepo
}

// Helper to create a service with mocks
func newServiceWithMocks() (*AccountService, *MockAccountRepo, *MockTransactionRepo) {
	accountRepo := &MockAccountRepo{}
	transactionRepo := &MockTransactionRepo{}
	uow := &MockUoW{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
	svc := NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil })
	return svc, accountRepo, transactionRepo
}

func TestCreateAccount_Success(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	accountRepo.On("Create", mock.Anything).Return(nil)

	userID := uuid.New()
	account, err := svc.CreateAccount(userID)
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, userID, account.UserID)
}

func TestCreateAccount_RepoError(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	accountRepo.On("Create", mock.Anything).Return(errors.New("db error"))

	userID := uuid.New()
	account, err := svc.CreateAccount(userID)
	assert.Error(t, err)
	assert.Nil(t, account)
}

func TestDeposit_Success(t *testing.T) {
	svc, accountRepo, transactionRepo := newServiceWithMocks()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.account = account
	accountRepo.On("Get", account.ID).Return(account, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)

	tx, err := svc.Deposit(account.ID, 100.0)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.InDelta(t, 100.0, account.GetBalance(), 0.01)
}

func TestDeposit_AccountNotFound(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	tx, err := svc.Deposit(uuid.New(), 100.0)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestDeposit_NegativeAmount(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.account = account
	accountRepo.On("Get", account.ID).Return(account, nil)

	tx, err := svc.Deposit(account.ID, -50.0)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrTransactionAmountMustBePositive, err)
}

func TestWithdraw_Success(t *testing.T) {
	svc, accountRepo, transactionRepo := newServiceWithMocks()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.account = account
	// Deposit first
	account.Deposit(100.0)
	accountRepo.On("Get", account.ID).Return(account, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)

	tx, err := svc.Withdraw(account.ID, 50.0)
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.InDelta(t, 50.0, account.GetBalance(), 0.01)
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.account = account
	accountRepo.On("Get", account.ID).Return(account, nil)

	tx, err := svc.Withdraw(account.ID, 100.0)
	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Equal(t, domain.ErrInsufficientFunds, err)
}

func TestGetAccount_Success(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.account = account
	accountRepo.On("Get", account.ID).Return(account, nil)

	got, err := svc.GetAccount(account.ID)
	assert.NoError(t, err)
	assert.Equal(t, account, got)
}

func TestGetAccount_NotFound(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	got, err := svc.GetAccount(uuid.New())
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Equal(t, domain.ErrAccountNotFound, err)
}

func TestGetTransactions_Success(t *testing.T) {
	svc, _, transactionRepo := newServiceWithMocks()
	accountID := uuid.New()
	txList := []*domain.Transaction{
		{ID: uuid.New(), AccountID: accountID, Amount: 100, Balance: 100},
	}
	transactionRepo.On("List", accountID).Return(txList, nil)

	got, err := svc.GetTransactions(accountID)
	assert.NoError(t, err)
	assert.Equal(t, txList, got)
}

func TestGetTransactions_Error(t *testing.T) {
	svc, _, transactionRepo := newServiceWithMocks()
	accountID := uuid.New()
	transactionRepo.On("List", accountID).Return([]*domain.Transaction{}, errors.New("db error"))

	got, err := svc.GetTransactions(accountID)
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetBalance_Success(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	account.Deposit(123.45)
	accountRepo.account = account
	accountRepo.On("Get", account.ID).Return(account, nil)

	balance, err := svc.GetBalance(account.ID)
	assert.NoError(t, err)
	assert.InDelta(t, 123.45, balance, 0.01)
}

func TestGetBalance_NotFound(t *testing.T) {
	svc, accountRepo, _ := newServiceWithMocks()
	accountRepo.On("Get", mock.Anything).Return(&domain.Account{}, domain.ErrAccountNotFound)

	balance, err := svc.GetBalance(uuid.New())
	assert.Error(t, err)
	assert.Equal(t, 0.0, balance)
}
