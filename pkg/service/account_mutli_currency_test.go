package service_test

import (
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeposit_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, nil, slog.Default())
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Create an account in EUR
	account, err := account.New().WithUserID(uuid.New()).WithCurrency("EUR").Build()
	assert.NoError(t, err)
	uow.EXPECT().AccountRepository().Return(repo, nil)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	// Try to deposit EUR
	gotTx, _, err := accountSvc.Deposit(account.UserID, account.ID, 100.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.Equal(t, currency.Code("EUR"), gotTx.Currency)
}

func TestWithdraw_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, nil, slog.Default())
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Create an account in EUR and deposit some funds
	account, _ := account.New().WithUserID(uuid.New()).WithCurrency("EUR").Build()
	m, err := money.NewMoney(100.0, "EUR")
	require.NoError(t, err)
	_, _ = account.Deposit(account.UserID, m)
	uow.EXPECT().AccountRepository().Return(repo, nil)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	// Try to withdraw EUR
	gotTx, _, err := accountSvc.Withdraw(account.UserID, account.ID, 50.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.Equal(t, currency.Code("EUR"), gotTx.Currency)
}

func TestDeposit_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Mock converter: 100 EUR -> 120 USD
	mockConverter := fixtures.NewMockCurrencyConverter(t)
	mockConverter.On("Convert", 100.0, "EUR", "USD").Return(&common.ConversionInfo{
		OriginalAmount:    100,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   120,
		ConvertedCurrency: "USD",
		ConversionRate:    1.2,
	}, nil)

	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, mockConverter, slog.Default())

	// Create an account in USD
	account, _ := account.New().WithUserID(uuid.New()).WithCurrency("USD").Build()
	uow.EXPECT().AccountRepository().Return(repo, nil)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	gotTx, _, err := accountSvc.Deposit(account.UserID, account.ID, 100.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.Equal(t, currency.Code("USD"), gotTx.Currency)
	mockConverter.AssertCalled(t, "Convert", 100.0, "EUR", "USD")
}

func TestWithdraw_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Mock converter: 50 EUR -> 60 USD
	mockConverter := fixtures.NewMockCurrencyConverter(t)
	mockConverter.On("Convert", 50.0, "EUR", "USD").Return(&common.ConversionInfo{
		OriginalAmount:    50,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   60,
		ConvertedCurrency: "USD",
		ConversionRate:    1.2,
	}, nil)

	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, mockConverter, slog.Default())

	// Create an account in USD and deposit some funds
	account, _ := account.New().WithUserID(uuid.New()).WithCurrency("USD").Build()
	m, err := money.NewMoney(100.0, "USD")
	require.NoError(t, err)
	_, _ = account.Deposit(account.UserID, m)
	uow.EXPECT().AccountRepository().Return(repo, nil)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	gotTx, _, err := accountSvc.Withdraw(account.UserID, account.ID, 50.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.Equal(t, currency.Code("USD"), gotTx.Currency)
	mockConverter.AssertCalled(t, "Convert", 50.0, "EUR", "USD")
}
