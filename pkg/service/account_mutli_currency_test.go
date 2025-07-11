package service_test

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeposit_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()

	account, _ := account.New().
		WithUserID(uuid.New()).
		WithBalance(10000).
		WithCurrency(currency.EUR).
		Build()
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	svc := service.NewAccountService(uow, nil, slog.Default())
	gotTx, _, err := svc.Deposit(account.UserID, account.ID, 100.0, currency.Code("EUR"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}

func TestWithdraw_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()

	account, _ := account.New().
		WithUserID(uuid.New()).
		WithBalance(10000).
		WithCurrency(currency.EUR).
		Build()
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	svc := service.NewAccountService(uow, nil, slog.Default())
	gotTx, _, err := svc.Withdraw(account.UserID, account.ID, 100.0, currency.Code("EUR"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}

func TestDeposit_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	converter := fixtures.NewMockCurrencyConverter(t)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()

	account := &account.Account{ID: uuid.New(), UserID: uuid.New(), Currency: currency.Code("USD"), Balance: 0}
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	// Fix: Return the correct type for converter.EXPECT().Convert
	converter.EXPECT().Convert(100.0, "EUR", "USD").Return(&common.ConversionInfo{
		OriginalAmount:    100.0,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   110.0,
		ConvertedCurrency: "USD",
		ConversionRate:    1.1,
	}, nil).Once()

	svc := service.NewAccountService(uow, converter, slog.Default())
	gotTx, _, err := svc.Deposit(account.UserID, account.ID, 100.0, currency.Code("EUR"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}

func TestWithdraw_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	converter := fixtures.NewMockCurrencyConverter(t)

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.AccountRepository)(nil)).Elem()).Return(accountRepo, nil).Once()
	uow.EXPECT().GetRepository(reflect.TypeOf((*repository.TransactionRepository)(nil)).Elem()).Return(transactionRepo, nil).Once()

	account, _ := account.New().
		WithUserID(uuid.New()).
		WithBalance(1000000).
		WithCurrency(currency.USD).
		Build()
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything).Return(nil).Once()

	converter.EXPECT().Convert(100.0, "EUR", "USD").Return(&common.ConversionInfo{
		OriginalAmount:    100.0,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   110.0,
		ConvertedCurrency: "USD",
		ConversionRate:    1.1,
	}, nil).Once()

	svc := service.NewAccountService(uow, converter, slog.Default())
	gotTx, _, err := svc.Withdraw(account.UserID, account.ID, 100.0, currency.Code("EUR"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}
