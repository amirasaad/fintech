package testutils

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
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
	// DefaultCurrencyCode is the default currency code used in tests
	DefaultCurrencyCode = "USD"
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
	Amount        *money.Money
	FeeAmount     *money.Money
}

// New creates a new test helper with fresh mocks and test data
func New(t *testing.T, opts ...TestOption) *TestHelper {
	t.Helper()

	// Setup defaults
	h := &TestHelper{
		T:      t,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), // Create a new logger for each test
	}

	// Initialize mocks
	h.UOW = mocks.NewUnitOfWork(t)
	h.MockAccRepo = mocks.NewAccountRepository(t)
	h.MockTxRepo = mocks.NewTransactionRepository(t)
	h.MockPaymentProvider = mocks.NewPaymentProvider(t)

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

	// Initialize test data with default values if not set
	if h.UserID == uuid.Nil {
		h.UserID = uuid.New()
	}

	if h.AccountID == uuid.Nil {
		h.AccountID = uuid.New()
	}

	if h.TransactionID == uuid.Nil {
		h.TransactionID = uuid.New()
	}

	if h.EventID == uuid.Nil {
		h.EventID = uuid.New()
	}

	if h.CorrelationID == uuid.Nil {
		h.CorrelationID = uuid.New()
	}

	// Initialize amounts if not set
	if h.Amount == nil {
		amount, err := money.New(DefaultAmount, money.Code(DefaultCurrencyCode).ToCurrency())
		require.NoError(t, err, "failed to create default amount")
		h.Amount = amount
	}

	if h.FeeAmount == nil {
		feeAmount, err := money.New(DefaultFeeAmount, money.Code(DefaultCurrencyCode).ToCurrency())
		require.NoError(t, err, "failed to create default fee amount")
		h.FeeAmount = feeAmount
	}

	return h
}

// TestOption defines a function type for test options
type TestOption func(*TestHelper)

var (
	initOnce sync.Once
)

var defaultTestOptions = []TestOption{
	// Initialize currency registry with default currencies
	func(h *TestHelper) {
		// Use sync.Once to ensure initialization happens only once
		initOnce.Do(func() {
			// No need to initialize currency registry as it's handled by the money package
		})
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
func (h *TestHelper) WithAmount(amount *money.Money) *TestHelper {
	h.Amount = amount
	return h
}

// WithFeeAmount sets a custom fee amount for the test helper
func (h *TestHelper) WithFeeAmount(amount *money.Money) *TestHelper {
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
	currency := h.Amount.CurrencyCode().String()
	return &dto.TransactionRead{
		ID:        h.TransactionID,
		UserID:    h.UserID,
		AccountID: h.AccountID,
		PaymentID: h.PaymentID,
		Status:    string(account.TransactionStatusPending),
		Currency:  currency,
		Amount:    amount,
	}
}

// CreateValidAccount creates a test account DTO
func (h *TestHelper) CreateValidAccount() *dto.AccountRead {
	amount := h.Amount.AmountFloat()
	currency := h.Amount.CurrencyCode().String()
	return &dto.AccountRead{
		ID:       h.AccountID,
		UserID:   h.UserID,
		Balance:  amount,
		Currency: currency,
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
