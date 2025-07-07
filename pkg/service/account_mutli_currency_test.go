package service_test

import (
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
