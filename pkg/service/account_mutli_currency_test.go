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

func TestDeposit_RejectsMismatchedCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t) // You'd use your actual mock here
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Rollback().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil })

	// Create an account in USD
	account := domain.NewAccountWithCurrency(uuid.New(), "USD")
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)

	// Try to deposit EUR
	_, err := accountSvc.DepositWithCurrency(account.UserID, account.ID, 100.0, "EUR")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency mismatch")
}

func TestDeposit_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil })

	// Create an account in EUR
	account := domain.NewAccountWithCurrency(uuid.New(), "EUR")
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	// Try to deposit EUR
	tx, err := accountSvc.DepositWithCurrency(account.UserID, account.ID, 100.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "EUR", tx.Currency)
}

func TestWithdraw_RejectsMismatchedCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Rollback().Return(nil)
	accountRepo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil })

	// Create an account in USD
	account := domain.NewAccountWithCurrency(uuid.New(), "USD")
	_, _ = account.DepositWithCurrency(account.UserID, 100.0, "USD")
	uow.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil)

	// Try to withdraw EUR
	_, err := accountSvc.WithdrawWithCurrency(account.UserID, account.ID, 50.0, "EUR")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency mismatch")
}

func TestWithdraw_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().Begin().Return(nil)
	uow.EXPECT().Commit().Return(nil)
	repo := fixtures.NewMockAccountRepository(t)
	accountSvc := service.NewAccountService(func() (repository.UnitOfWork, error) { return uow, nil })
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	uow.EXPECT().TransactionRepository().Return(transactionRepo)
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil)

	// Create an account in EUR and deposit some funds
	account := domain.NewAccountWithCurrency(uuid.New(), "EUR")
	account.DepositWithCurrency(account.UserID, 100.0, "EUR")
	uow.EXPECT().AccountRepository().Return(repo)
	repo.EXPECT().Get(account.ID).Return(account, nil)
	repo.EXPECT().Update(account).Return(nil)

	// Try to withdraw EUR
	tx, err := accountSvc.WithdrawWithCurrency(account.UserID, account.ID, 50.0, "EUR")
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "EUR", tx.Currency)
}
