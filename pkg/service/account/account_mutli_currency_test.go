package account_test

import (
	"context"
	"log/slog"
	"testing"

	fixtures "github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/repository"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeposit_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)

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

	account, _ := accountdomain.New().
		WithUserID(uuid.New()).
		WithBalance(10000).
		WithCurrency(currency.EUR).
		Build()
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotTx, _, err := svc.Deposit(account.UserID, account.ID, 100.0, currency.Code("EUR"), "Cash")
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}

func TestWithdraw_AcceptsMatchingCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)

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

	account, _ := accountdomain.New().
		WithUserID(uuid.New()).
		WithBalance(10000).
		WithCurrency(currency.EUR).
		Build()
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()

	svc := accountsvc.NewAccountService(uow, nil, slog.Default())
	gotTx, _, err := svc.Withdraw(account.UserID, account.ID, 100.0, currency.Code("EUR"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}

func TestDeposit_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	converter := fixtures.NewMockCurrencyConverter(t)

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

	account := &accountdomain.Account{ID: uuid.New(), UserID: uuid.New(), Balance: money.Zero(currency.USD)}
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()

	// Fix: Return the correct type for converter.EXPECT().Convert
	converter.EXPECT().Convert(100.0, "EUR", "USD").Return(&common.ConversionInfo{
		OriginalAmount:    100.0,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   110.0,
		ConvertedCurrency: "USD",
		ConversionRate:    1.1,
	}, nil).Once()

	svc := accountsvc.NewAccountService(uow, converter, slog.Default())
	gotTx, _, err := svc.Deposit(account.UserID, account.ID, 100.0, currency.Code("EUR"), "Cash")
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}

func TestWithdraw_ConvertsCurrency(t *testing.T) {
	uow := fixtures.NewMockUnitOfWork(t)
	accountRepo := fixtures.NewMockAccountRepository(t)
	transactionRepo := fixtures.NewMockTransactionRepository(t)
	converter := fixtures.NewMockCurrencyConverter(t)

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

	account, _ := accountdomain.New().
		WithUserID(uuid.New()).
		WithBalance(1000000).
		WithCurrency(currency.USD).
		Build()
	accountRepo.EXPECT().Get(account.ID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()

	converter.EXPECT().Convert(100.0, "EUR", "USD").Return(&common.ConversionInfo{
		OriginalAmount:    100.0,
		OriginalCurrency:  "EUR",
		ConvertedAmount:   110.0,
		ConvertedCurrency: "USD",
		ConversionRate:    1.1,
	}, nil).Once()

	svc := accountsvc.NewAccountService(uow, converter, slog.Default())
	gotTx, _, err := svc.Withdraw(account.UserID, account.ID, 100.0, currency.Code("EUR"))
	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
}
