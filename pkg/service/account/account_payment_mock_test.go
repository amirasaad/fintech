package account_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/amirasaad/fintech/infra/provider"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	accountdomain "github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/repository"
	accountsvc "github.com/amirasaad/fintech/pkg/service/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDepositWithMockPaymentProvider(t *testing.T) {
	// Arrange: create mock dependencies (replace with real or mock implementations as needed)
	uow, accountRepo, transactionRepo := setupTestMocks(t)
	mockConverter := mocks.NewMockCurrencyConverter(t)
	mockLogger := slog.Default()
	mockPayment := provider.NewMockPaymentProvider()

	userID := uuid.New()
	accountID := uuid.New()
	account, _ := accountdomain.New().WithUserID(userID).WithBalance(100).Build()
	amount := 100.0
	curr := "USD"

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

	accountRepo.EXPECT().Get(accountID).Return(account, nil).Once()
	accountRepo.EXPECT().Update(mock.Anything).Return(nil).Once()
	transactionRepo.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	svc := accountsvc.NewService(accountsvc.ServiceDeps{
		Uow:             uow,
		Converter:       mockConverter,
		Logger:          mockLogger,
		PaymentProvider: mockPayment,
	})

	ctx := context.Background()

	// Act: initiate deposit (simulate payment)
	paymentID, err := mockPayment.InitiateDeposit(ctx, userID, accountID, amount, curr)
	require.NoError(t, err)
	require.NotEmpty(t, paymentID)

	// Simulate waiting for payment provider to complete
	time.Sleep(3 * time.Second)

	status, err := mockPayment.GetPaymentStatus(ctx, paymentID)
	require.NoError(t, err)
	require.Equal(t, provider.PaymentCompleted, status)

	// Optionally, call your service's Deposit method and assert results
	tx, _, err := svc.Deposit(userID, accountID, amount, currency.Code(curr), "Cash")
	require.NoError(t, err)
	require.NotNil(t, tx)
}
