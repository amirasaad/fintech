package account

import (
	"context"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test BaseHandler
func TestBaseHandler_SetNext(t *testing.T) {
	base := &BaseHandler{}
	next := &BaseHandler{}
	base.SetNext(next)
	assert.Equal(t, next, base.next)
}

func TestBaseHandler_Handle_WithNoNext(t *testing.T) {
	base := &BaseHandler{}
	req := &OperationRequest{}
	resp, err := base.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, resp.Transaction)
	assert.Nil(t, resp.ConvInfo)
	assert.NoError(t, resp.Error)
}

// Mock handler for testing
// Only used for BaseHandler chain test
type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OperationResponse), args.Error(1)
}

func (m *MockHandler) SetNext(handler OperationHandler) {
	m.Called(handler)
}

func TestBaseHandler_Handle_WithNext(t *testing.T) {
	base := &BaseHandler{}
	next := &MockHandler{}
	req := &OperationRequest{}
	expectedResp := &OperationResponse{Transaction: &account.Transaction{}}
	next.On("Handle", context.Background(), req).Return(expectedResp, nil)
	base.SetNext(next)
	resp, err := base.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
	next.AssertExpectations(t)
}

// Test AccountValidationHandler
func TestAccountValidationHandler_Handle_Success(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	logger := newTestLogger()
	userID := uuid.New()
	accountID := uuid.New()
	req := &OperationRequest{
		UserID:    userID,
		AccountID: accountID,
	}
	expectedAccount := &account.Account{
		ID:     accountID,
		UserID: userID,
	}
	uow.On("AccountRepository").Return(accountRepo, nil)
	accountRepo.On("Get", accountID).Return(expectedAccount, nil)
	handler := &ValidationHandler{
		uow:    uow,
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.Equal(t, expectedAccount, req.Account)
	uow.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
}

func TestAccountValidationHandler_Handle_RepositoryError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	logger := newTestLogger()
	req := &OperationRequest{
		UserID:    uuid.New(),
		AccountID: uuid.New(),
	}
	uow.On("AccountRepository").Return(nil, assert.AnError)
	handler := &ValidationHandler{
		uow:    uow,
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Error(t, resp.Error)
	assert.Equal(t, assert.AnError, resp.Error)
	uow.AssertExpectations(t)
}

func TestAccountValidationHandler_Handle_AccountNotFound(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	logger := newTestLogger()
	userID := uuid.New()
	accountID := uuid.New()
	req := &OperationRequest{
		UserID:    userID,
		AccountID: accountID,
	}
	uow.On("AccountRepository").Return(accountRepo, nil)
	accountRepo.On("Get", accountID).Return(nil, account.ErrAccountNotFound)
	handler := &ValidationHandler{
		uow:    uow,
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Error(t, resp.Error)
	assert.Equal(t, account.ErrAccountNotFound, resp.Error)
	uow.AssertExpectations(t)
	accountRepo.AssertExpectations(t)
}

// Test MoneyCreationHandler
func TestMoneyCreationHandler_Handle_Success(t *testing.T) {
	logger := newTestLogger()
	req := &OperationRequest{
		Amount:       100.0,
		CurrencyCode: currency.Code("USD"),
	}
	handler := &MoneyCreationHandler{
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.NotNil(t, req.Money)
	assert.Equal(t, float64(100.0), req.Money.AmountFloat())
	assert.Equal(t, currency.Code("USD"), req.Money.Currency())
}

func TestMoneyCreationHandler_Handle_InvalidMoney(t *testing.T) {
	logger := newTestLogger()
	req := &OperationRequest{
		Amount:       -100.0, // Negative amount is valid for Money creation
		CurrencyCode: currency.Code("USD"),
	}
	handler := &MoneyCreationHandler{
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error) // Money creation succeeds even for negative amounts
	assert.NotNil(t, req.Money)
	assert.Equal(t, float64(-100.0), req.Money.AmountFloat())
	assert.Equal(t, currency.Code("USD"), req.Money.Currency())
}

// Test CurrencyConversionHandler
func TestCurrencyConversionHandler_Handle_NoConversionNeeded(t *testing.T) {
	converter := mocks.NewMockCurrencyConverter(t)
	logger := newTestLogger()
	money, _ := money.NewMoney(100.0, currency.Code("USD"))
	req := &OperationRequest{
		Money: money,
		Account: &account.Account{
			Currency: currency.Code("USD"),
		},
	}
	handler := &CurrencyConversionHandler{
		converter: converter,
		logger:    logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.Equal(t, money, req.ConvertedMoney)
	assert.Nil(t, req.ConvInfo)
}

func TestCurrencyConversionHandler_Handle_ConversionNeeded(t *testing.T) {
	converter := mocks.NewMockCurrencyConverter(t)
	logger := newTestLogger()
	money, _ := money.NewMoney(100.0, currency.Code("USD"))
	req := &OperationRequest{
		Money: money,
		Account: &account.Account{
			Currency: currency.Code("EUR"),
		},
	}
	convInfo := &common.ConversionInfo{
		OriginalAmount:    100.0,
		ConvertedAmount:   85.0,
		OriginalCurrency:  "USD",
		ConvertedCurrency: "EUR",
		ConversionRate:    0.85,
	}
	converter.On("Convert", 100.0, "USD", "EUR").Return(convInfo, nil)
	handler := &CurrencyConversionHandler{
		converter: converter,
		logger:    logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.NotNil(t, req.ConvertedMoney)
	assert.Equal(t, float64(85.0), req.ConvertedMoney.AmountFloat())
	assert.Equal(t, currency.Code("EUR"), req.ConvertedMoney.Currency())
	assert.Equal(t, convInfo, req.ConvInfo)
	converter.AssertExpectations(t)
}

func TestCurrencyConversionHandler_Handle_ConversionError(t *testing.T) {
	converter := mocks.NewMockCurrencyConverter(t)
	logger := newTestLogger()
	money, _ := money.NewMoney(100.0, currency.Code("USD"))
	req := &OperationRequest{
		Money: money,
		Account: &account.Account{
			Currency: currency.Code("EUR"),
		},
	}
	converter.On("Convert", 100.0, "USD", "EUR").Return(nil, assert.AnError)
	handler := &CurrencyConversionHandler{
		converter: converter,
		logger:    logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Error(t, resp.Error)
	assert.Equal(t, assert.AnError, resp.Error)
	converter.AssertExpectations(t)
}

// Test DomainOperationHandler
func TestDomainOperationHandler_Handle_Deposit(t *testing.T) {
	logger := newTestLogger()
	money, _ := money.NewMoney(100.0, currency.Code("USD"))
	userID := uuid.New()
	account := &account.Account{
		ID:       uuid.New(),
		UserID:   userID, // Set the correct user ID
		Currency: currency.Code("USD"),
	}
	req := &OperationRequest{
		UserID:         userID, // Use the same user ID
		Operation:      OperationDeposit,
		ConvertedMoney: money,
		Account:        account,
	}
	handler := &DomainOperationHandler{
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.NotNil(t, req.Transaction)
	assert.Equal(t, account.ID, req.Transaction.AccountID)
}

func TestDomainOperationHandler_Handle_Withdraw(t *testing.T) {
	logger := newTestLogger()
	money, _ := money.NewMoney(100.0, currency.Code("USD"))
	userID := uuid.New()
	account := &account.Account{
		ID:       uuid.New(),
		UserID:   userID, // Set the correct user ID
		Currency: currency.Code("USD"),
		Balance:  10000, // Set sufficient balance (100.00 USD in cents)
	}
	req := &OperationRequest{
		UserID:         userID, // Use the same user ID
		Operation:      OperationWithdraw,
		ConvertedMoney: money,
		Account:        account,
	}
	handler := &DomainOperationHandler{
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.NotNil(t, req.Transaction)
	assert.Equal(t, account.ID, req.Transaction.AccountID)
}

func TestDomainOperationHandler_Handle_UnsupportedOperation(t *testing.T) {
	logger := newTestLogger()
	req := &OperationRequest{
		Operation: "invalid",
	}
	handler := &DomainOperationHandler{
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Error(t, resp.Error)
	assert.Contains(t, resp.Error.Error(), "unsupported operation")
}

// Test PersistenceHandler
func TestPersistenceHandler_Handle_Success(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	txRepo := mocks.NewMockTransactionRepository(t)
	logger := newTestLogger()
	transaction := &account.Transaction{
		ID:        uuid.New(),
		AccountID: uuid.New(),
	}
	account := &account.Account{
		ID: uuid.New(),
	}
	convInfo := &common.ConversionInfo{
		OriginalAmount: 100.0,
		ConversionRate: 0.85,
	}
	req := &OperationRequest{
		Transaction: transaction,
		Account:     account,
		ConvInfo:    convInfo,
	}

	// Do called by PersistenceHandler
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	// Inside Do callback: AccountRepository and TransactionRepository called once each
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()
	uow.EXPECT().TransactionRepository().Return(txRepo, nil).Once()

	accountRepo.EXPECT().Update(account).Return(nil).Once()
	txRepo.EXPECT().Create(transaction).Return(nil).Once()

	handler := &PersistenceHandler{
		uow:    uow,
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
	assert.Equal(t, transaction, resp.Transaction)
	assert.Equal(t, convInfo, resp.ConvInfo)
	assert.Equal(t, &convInfo.OriginalAmount, transaction.OriginalAmount)
	assert.Equal(t, &convInfo.ConversionRate, transaction.ConversionRate)
}

func TestPersistenceHandler_Handle_AccountUpdateError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	logger := newTestLogger()
	transaction := &account.Transaction{ID: uuid.New()}
	account := &account.Account{ID: uuid.New()}
	req := &OperationRequest{
		Transaction: transaction,
		Account:     account,
	}

	// Do called by PersistenceHandler
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	// Inside Do callback: AccountRepository called once
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()

	accountRepo.EXPECT().Update(account).Return(assert.AnError).Once()

	handler := &PersistenceHandler{
		uow:    uow,
		logger: logger,
	}
	resp, err := handler.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.Error(t, resp.Error)
	assert.Equal(t, assert.AnError, resp.Error)
}

// Test ChainBuilder
func TestChainBuilder_BuildOperationChain(t *testing.T) {
	userId := uuid.New()
	acc, _ := account.New().WithUserID(userId).Build()
	uow := mocks.NewMockUnitOfWork(t)
	accountRepo := mocks.NewMockAccountRepository(t)
	transactionRepo := mocks.NewMockTransactionRepository(t)
	converter := mocks.NewMockCurrencyConverter(t)

	// AccountRepository called once by AccountValidationHandler
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()

	// Do called once by PersistenceHandler
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()

	// Inside Do callback: AccountRepository and TransactionRepository called once each
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Once()
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Once()

	accountRepo.EXPECT().Get(mock.Anything).Return(acc, nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()
	accountRepo.EXPECT().Update(acc).Return(nil).Once()

	logger := newTestLogger()
	builder := NewChainBuilder(uow, converter, logger)
	chain := builder.BuildOperationChain()
	assert.NotNil(t, chain)
	req := &OperationRequest{
		UserID:       userId,
		AccountID:    acc.ID,
		Amount:       100.0,
		CurrencyCode: currency.Code("USD"),
		Operation:    OperationDeposit,
	}
	resp, err := chain.Handle(context.Background(), req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Error)
}

// Helper function to create a test logger
func newTestLogger() *slog.Logger {
	return slog.Default()
}
