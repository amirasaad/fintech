package service

import (
	"math"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateAccount(b *testing.B) {
	svc, accountRepo, _, uow := newServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	uow.EXPECT().AccountRepository().Return(accountRepo).Maybe()
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
	uow.EXPECT().AccountRepository().Return(accountRepo).Maybe()
	uow.EXPECT().TransactionRepository().Return(transactionRepo).Maybe()
	userID := uuid.New()
	account := domain.NewAccount(userID)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Maybe()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Maybe()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = svc.Deposit(userID, account.ID, 100.0, "USD")
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
	account.Balance = int64(math.MaxInt64)
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Maybe()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Maybe()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = svc.Withdraw(userID, account.ID, 50.0, "USD")
	}
}
