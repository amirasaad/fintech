package service

import (
	"context"
	"math"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateAccount(b *testing.B) {
	svc, accountRepo, _, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	accountRepo.On("Create", mock.Anything).Return(nil)
	userID := uuid.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CreateAccount(context.Background(), userID)
	}
}

func BenchmarkDeposit(b *testing.B) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	accountRepo.On("Get", acc.ID).Return(acc, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	accountRepo.On("Create", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = svc.Deposit(userID, acc.ID, 100.0, "USD")
	}
}

func BenchmarkWithdraw(b *testing.B) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	acc.Balance = int64(math.MaxInt64)
	accountRepo.On("Get", acc.ID).Return(acc, nil)
	accountRepo.On("Update", mock.Anything).Return(nil)
	transactionRepo.On("Create", mock.Anything).Return(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = svc.Withdraw(userID, acc.ID, 50.0, "USD")
	}
}
