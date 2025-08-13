package testutils

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/eventbus"
	"github.com/amirasaad/fintech/pkg/money"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	// DefaultCurrency is the default currency used in tests
	DefaultCurrency = "USD"
	// DefaultAmount is the default amount used in tests (100.00)
	DefaultAmount = 100.0
	// DefaultFeeAmount is the default fee amount used in tests (1.00)
	DefaultFeeAmount = 1.0
)

type TestEvent struct{}

func (e *TestEvent) Type() string { return "test.event" }

// TestHelper contains all test dependencies and helper methods
type TestHelper struct {
	T                   *testing.T
	Handler             eventbus.HandlerFunc
	MockPaymentProvider *mocks.PaymentProvider
	Ctx                 context.Context
	Bus                 *mocks.Bus
	UOW                 *mocks.UnitOfWork
	MockAccRepo         *mocks.AccountRepository
	MockTxRepo          *mocks.TransactionRepository
	Logger              *slog.Logger

	// Test data
	UserID        uuid.UUID
	AccountID     uuid.UUID
	PaymentID     string
	EventID       uuid.UUID
	CorrelationID uuid.UUID
	TransactionID uuid.UUID
	Amount        money.Money
	FeeAmount     money.Money
}

// New creates a new test helper with fresh mocks and test data
func New(t *testing.T, opts ...TestOption) *TestHelper {
	t.Helper()

	// Setup defaults
	h := &TestHelper{
		T:      t,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), // Create a new logger for each test
	}

	// Apply options (excluding default currency initialization since we did it above)
	for _, opt := range append(defaultTestOptions, opts...) {
		opt(h)
	}

	// Initialize test data if not set by options
	if h.Handler == nil {
		h.Handler = eventbus.HandlerFunc(
			func(ctx context.Context, event events.Event) error {
				return nil
			})
	}
	if h.Ctx == nil {
		h.Ctx = context.Background()
	}

	if h.Bus == nil {
		h.Bus = mocks.NewBus(t)
	}

	if h.UOW == nil {
		h.UOW = mocks.NewUnitOfWork(t)
	}

	if h.MockAccRepo == nil {
		h.MockAccRepo = mocks.NewAccountRepository(t)
	}

	if h.MockTxRepo == nil {
		h.MockTxRepo = mocks.NewTransactionRepository(t)
	}
	if h.MockPaymentProvider == nil {
		h.MockPaymentProvider = mocks.NewPaymentProvider(t)
		require.NotNil(t, h.MockPaymentProvider, "MockPaymentProvider should not be nil")
	}

	// Initialize test data
	if h.UserID == uuid.Nil {
		h.UserID = uuid.New()
	}

	if h.AccountID == uuid.Nil {
		h.AccountID = uuid.New()
	}

	if h.PaymentID == "" {
		h.PaymentID = "test_payment_" + uuid.New().String()
	}

	if h.EventID == uuid.Nil {
		h.EventID = uuid.New()
	}

	if h.CorrelationID == uuid.Nil {
		h.CorrelationID = uuid.New()
	}

	if h.TransactionID == uuid.Nil {
		h.TransactionID = uuid.New()
	}

	// Initialize amounts if not set
	if h.Amount.IsZero() {
		amount, err := money.New(DefaultAmount, DefaultCurrency)
		require.NoError(t, err, "failed to create default amount")
		h.Amount = amount
	}

	if h.FeeAmount.IsZero() {
		feeAmount, err := money.New(DefaultFeeAmount, DefaultCurrency)
		require.NoError(t, err, "failed to create default fee amount")
		h.FeeAmount = feeAmount
	}

	return h
}

// TestOption defines a function type for test options
type TestOption func(*TestHelper)

var defaultTestOptions = []TestOption{
	// Initialize currency registry with default currencies
	func(h *TestHelper) {
		// Initialize the global currency registry if not already done
		ctx := context.Background()
		err := currency.InitializeGlobalRegistry(ctx)
		if err != nil {
			h.T.Fatalf("failed to initialize global currency registry: %v", err)
		}
	},
}

// WithHandler sets a custom handler for the test helper
func (h *TestHelper) WithHandler(
	handler eventbus.HandlerFunc) *TestHelper {
	h.Handler = handler
	return h
}

// WithContext sets the context for the test helper
func (h *TestHelper) WithContext(ctx context.Context) *TestHelper {
	h.Ctx = ctx
	return h
}

// WithAmount sets a custom amount for the test helper
func (h *TestHelper) WithAmount(amount money.Money) *TestHelper {
	h.Amount = amount
	return h
}

// WithFeeAmount sets a custom fee amount for the test helper
func (h *TestHelper) WithFeeAmount(amount money.Money) *TestHelper {
	h.FeeAmount = amount
	return h
}

// WithUserID sets a custom user ID for the test helper
func (h *TestHelper) WithUserID(id uuid.UUID) *TestHelper {
	h.UserID = id
	return h
}

// WithAccountID sets a custom account ID for the test helper
func (h *TestHelper) WithAccountID(id uuid.UUID) *TestHelper {
	h.AccountID = id
	return h
}

// WithTransactionID sets a custom transaction ID for the test helper
func (h *TestHelper) WithTransactionID(d uuid.UUID) *TestHelper {
	h.TransactionID = d
	return h
}

// WithPaymentID sets a custom payment ID for the test helper
func (h *TestHelper) WithPaymentID(id string) *TestHelper {
	h.PaymentID = id
	return h
}

// CreateValidTransaction creates a test transaction DTO
func (h *TestHelper) CreateValidTransaction() *dto.TransactionRead {
	amount := h.Amount.AmountFloat()
	return &dto.TransactionRead{
		ID:        h.TransactionID,
		UserID:    h.UserID,
		AccountID: h.AccountID,
		PaymentID: h.PaymentID,
		Status:    string(account.TransactionStatusPending),
		Currency:  DefaultCurrency,
		Amount:    amount,
	}
}

// CreateValidAccount creates a test account DTO
func (h *TestHelper) CreateValidAccount() *dto.AccountRead {
	amount := h.Amount.AmountFloat()
	return &dto.AccountRead{
		ID:       h.AccountID,
		UserID:   h.UserID,
		Balance:  amount,
		Currency: DefaultCurrency,
	}
}

// SetupMocks configures the default mock expectations
func (h *TestHelper) SetupMocks() {
	h.UOW.EXPECT().GetRepository(mock.Anything).Return(h.MockAccRepo, nil).Maybe()
	h.UOW.EXPECT().GetRepository(mock.Anything).Return(h.MockTxRepo, nil).Maybe()
}

// AssertExpectations asserts all mock expectations
func (h *TestHelper) AssertExpectations() {
}
