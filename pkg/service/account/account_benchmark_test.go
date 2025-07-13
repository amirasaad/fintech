package account_test

import (
	"context"
	"math"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func BenchmarkCreateAccount(b *testing.B) {
	require := require.New(b)
	svc, accountRepo, _, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	accountRepo.On("Create", mock.Anything).Return(nil)
	userID := uuid.New()
	b.ResetTimer()
	for b.Loop() {
		_, err := svc.CreateAccount(context.Background(), userID)
		require.NoError(err)

	}
}

func BenchmarkDeposit(b *testing.B) {
	require := require.New(b)
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	userID := uuid.New()
	acc, _ := accountdomain.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	accountRepo.On("Get", acc.ID).Return(acc, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	b.ResetTimer()
	for b.Loop() {
		_, _, err := svc.Deposit(userID, acc.ID, 100.0, "USD")
		require.NoError(err)
	}
}

func BenchmarkWithdraw(b *testing.B) {
	require := require.New(b)
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	userID := uuid.New()
	acc, _ := accountdomain.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	acc.Balance, _ = money.New(math.MaxInt64, acc.Balance.Currency())
	accountRepo.On("Get", acc.ID).Return(acc, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	b.ResetTimer()
	for b.Loop() {
		_, _, err := svc.Withdraw(userID, acc.ID, 50.0, "USD")
		require.NoError(err)
	}
}
