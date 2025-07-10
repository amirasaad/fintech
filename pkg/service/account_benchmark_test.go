package service

import (
	"math"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateAccount(b *testing.B) {
	svc, accountRepo, _, uow := newServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Maybe()
	accountRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
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
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Maybe()
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Maybe()
	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil).Maybe()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Maybe()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = svc.Deposit(userID, acc.ID, 100.0, "USD")
	}
}

func BenchmarkWithdraw(b *testing.B) {
	svc, accountRepo, transactionRepo, uow := newServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	uow.EXPECT().AccountRepository().Return(accountRepo, nil).Maybe()
	uow.EXPECT().TransactionRepository().Return(transactionRepo, nil).Maybe()
	userID := uuid.New()
	acc, _ := account.New().WithUserID(userID).WithCurrency(currency.USD).Build() //nolint:errcheck
	acc.Balance = int64(math.MaxInt64)
	accountRepo.EXPECT().Get(acc.ID).Return(acc, nil).Maybe()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Maybe()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = svc.Withdraw(userID, acc.ID, 50.0, "USD")
	}
}
