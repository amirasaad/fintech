package payment

import (
	"context"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/repository"
	repoaccount "github.com/amirasaad/fintech/pkg/repository/account"
	repotransaction "github.com/amirasaad/fintech/pkg/repository/transaction"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testHelper contains all test dependencies and helper methods
type testHelper struct {
	t           *testing.T
	ctx         context.Context
	bus         eventbus.Bus
	uow         *mocks.UnitOfWork
	mockAccRepo *mocks.AccountRepository
	mockTxRepo  *mocks.TransactionRepository
	logger      *slog.Logger

	// Test data
	userID        uuid.UUID
	accountID     uuid.UUID
	paymentID     string
	eventID       uuid.UUID
	correlationID uuid.UUID
	transactionID uuid.UUID
	amount        money.Money
	feeAmount     money.Money
}

// newTestHelper creates a new test helper with fresh mocks and test data
func newTestHelper(t *testing.T) *testHelper {
	t.Helper()
	ctx,
		bus,
		uow,
		userID,
		accountID,
		paymentID,
		transactionID,
		eventID,
		correlationID := setupTest(t)
	h := &testHelper{
		t:             t,
		ctx:           ctx,
		bus:           bus,
		uow:           uow,
		mockAccRepo:   mocks.NewAccountRepository(t),
		mockTxRepo:    mocks.NewTransactionRepository(t),
		logger:        slog.Default(),
		userID:        userID,
		accountID:     accountID,
		paymentID:     paymentID,
		eventID:       eventID,
		correlationID: correlationID,
		transactionID: transactionID,
	}

	var err error
	h.amount, err = money.New(10000, "USD") // $100.00
	require.NoError(t, err, "failed to create amount")

	h.feeAmount, err = money.New(100, "USD") // $1.00
	require.NoError(t, err, "failed to create fee amount")

	return h
}

// createValidEvent creates a valid PaymentCompletedEvent
func (h *testHelper) createValidEvent() *events.PaymentCompleted {
	return &events.PaymentCompleted{
		PaymentInitiated: events.PaymentInitiated{
			FlowEvent: events.FlowEvent{
				ID:            h.eventID,
				CorrelationID: h.correlationID,
				FlowType:      "payment",
			},
			Amount:    h.amount,
			PaymentID: h.paymentID,
			Status:    "succeeded",
		},
		ProviderFee: account.Fee{
			Amount: h.feeAmount,
			Type:   account.FeeProvider,
		},
	}
}

// setupSuccessfulTest configures mocks for a successful payment completion
func (h *testHelper) setupSuccessfulTest() {
	// Setup test transaction
	tx := &dto.TransactionRead{
		ID:        h.transactionID,
		UserID:    h.userID,
		AccountID: h.accountID,
		PaymentID: h.paymentID,
		Status:    string(account.TransactionStatusPending),
		Currency:  "USD",
		Amount:    h.amount.AmountFloat(),
	}

	// Setup test account
	testAccount := &dto.AccountRead{
		ID:       h.accountID,
		UserID:   h.userID,
		Balance:  h.amount.AmountFloat(), // $1000.00
		Currency: "USD",
	}

	// Setup mocks with correct method signatures
	h.mockTxRepo.
		EXPECT().
		GetByPaymentID(mock.Anything, h.paymentID).
		Return(tx, nil).Once()
	h.mockAccRepo.
		EXPECT().
		Get(mock.Anything, h.accountID).
		Return(testAccount, nil).Once()

	// Update the mock expectations to match the actual method signatures
	h.mockTxRepo.EXPECT().Update(
		mock.Anything,
		h.transactionID,
		mock.AnythingOfType("dto.TransactionUpdate"),
	).Return(nil).Once()
	h.mockAccRepo.EXPECT().Update(
		mock.Anything,
		h.accountID,
		mock.AnythingOfType("dto.AccountUpdate"),
	).Return(nil).Once()

	// Setup UOW to return the mock repositories
	h.uow.EXPECT().GetRepository(
		(*repoaccount.Repository)(nil),
	).Return(h.mockAccRepo, nil).Once()
	h.uow.EXPECT().GetRepository(
		(*repotransaction.Repository)(nil),
	).Return(h.mockTxRepo, nil).Once()

	// Mock the Do callback
	h.uow.
		EXPECT().
		Do(mock.Anything, mock.Anything).
		Return(nil).
		Run(func(
			ctx context.Context,
			fn func(uow repository.UnitOfWork) error,
		) {
			err := fn(h.uow)
			require.NoError(h.t, err)
		}).Once()
}

func setupTest(t *testing.T) (
	ctx context.Context,
	bus *mocks.Bus,
	mUow *mocks.UnitOfWork,
	userID uuid.UUID,
	accountID uuid.UUID,
	paymentID string,
	transactionID uuid.UUID,
	eventID uuid.UUID,
	correlationID uuid.UUID,
) {
	t.Helper()
	ctx = context.Background()
	bus = mocks.NewBus(t)
	mUow = mocks.NewUnitOfWork(t)

	// Generate consistent test IDs
	userID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	accountID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	paymentID = "pay_123"
	eventID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	correlationID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	transactionID = uuid.MustParse("55555555-5555-5555-5555-555555555555")

	return
}
