package service_test

import (
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeposit_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, nil)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Create an account in EUR
	account := domain.NewAccountWithCurrency(uuid.New(), "EUR")
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	// Try to deposit EUR
	tx, _, err := accountSvc.Deposit(account.UserID, account.ID, 100.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "EUR", tx.Currency)
}

func TestWithdraw_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, nil)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Create an account in EUR and deposit some funds
	account := domain.NewAccountWithCurrency(uuid.New(), "EUR")
	_, _ = account.Deposit(account.UserID, domain.Money{Amount: 100.0, Currency: "EUR"})
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	// Try to withdraw EUR
	tx, _, err := accountSvc.Withdraw(account.UserID, account.ID, 50.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "EUR", tx.Currency)
}

func TestDeposit_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Mock converter: 100 EUR -> 120 USD
	mockConverter := fixtures.NewMockCurrencyConverter(t)
	mockConverter.On("Convert", 100.0, "EUR", "USD").Return(&domain.ConversionInfo{
		OriginalAmount:    100,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   120,
		ConvertedCurrency: "USD",
		ConversionRate:    1.2,
	}, nil)

	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, mockConverter)

	// Create an account in USD
	account := domain.NewAccountWithCurrency(uuid.New(), "USD")
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	tx, _, err := accountSvc.Deposit(account.UserID, account.ID, 100.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "USD", tx.Currency)
	mockConverter.AssertCalled(t, "Convert", 100.0, "EUR", "USD")
}

func TestWithdraw_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Mock converter: 50 EUR -> 60 USD
	mockConverter := fixtures.NewMockCurrencyConverter(t)
	mockConverter.On("Convert", 50.0, "EUR", "USD").Return(&domain.ConversionInfo{
		OriginalAmount:    50,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   60,
		ConvertedCurrency: "USD",
		ConversionRate:    1.2,
	}, nil)

	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil }, mockConverter)

	// Create an account in USD and deposit some funds
	account := domain.NewAccountWithCurrency(uuid.New(), "USD")
	_, _ = account.Deposit(account.UserID, domain.Money{Amount: 100.0, Currency: "USD"})
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	tx, _, err := accountSvc.Withdraw(account.UserID, account.ID, 50.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "USD", tx.Currency)
	mockConverter.AssertCalled(t, "Convert", 50.0, "EUR", "USD")
}
