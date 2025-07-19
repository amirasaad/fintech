package conversion

import (
	"context"
	"testing"
	"time"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockCurrencyConverter implements money.CurrencyConverter for testing
type MockCurrencyConverter struct {
	mock.Mock
}

func (m *MockCurrencyConverter) Convert(amount float64, fromCurrency, toCurrency string) (*common.ConversionInfo, error) {
	args := m.Called(amount, fromCurrency, toCurrency)
	return args.Get(0).(*common.ConversionInfo), args.Error(1)
}

func (m *MockCurrencyConverter) GetRate(fromCurrency, toCurrency string) (float64, error) {
	args := m.Called(fromCurrency, toCurrency)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockCurrencyConverter) IsSupported(fromCurrency, toCurrency string) bool {
	args := m.Called(fromCurrency, toCurrency)
	return args.Bool(0)
}

// MockEventBus implements eventbus.EventBus for testing
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, event domain.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType string, handler func(context.Context, domain.Event)) {
	m.Called(eventType, handler)
}

func TestHandler_ConversionRequestedEvent(t *testing.T) {
	// Setup
	mockConverter := &MockCurrencyConverter{}
	mockBus := &MockEventBus{}
	logger := slog.Default()

	handler := Handler(mockBus, mockConverter, logger)

	// Create test event
	fromAmount, _ := money.New(100.0, "USD")
	event := events.ConversionRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     uuid.New(),
			UserID:        uuid.New(),
			CorrelationID: uuid.New(),
			FlowType:      "conversion",
		},
		FromAmount: fromAmount,
		ToCurrency: "EUR",
		RequestID:  "test-request-id",
	}

	// Setup expectations
	conversionInfo := &common.ConversionInfo{
		OriginalAmount:    100.0,
		OriginalCurrency:  "USD",
		ConvertedAmount:   85.0,
		ConvertedCurrency: "EUR",
		ConversionRate:    0.85,
	}
	mockConverter.On("Convert", 100.0, "USD", "EUR").Return(conversionInfo, nil)
	mockBus.On("Publish", mock.Anything, mock.AnythingOfType("events.ConversionDoneEvent")).Return(nil).Once()

	// Execute
	ctx := context.Background()
	handler(ctx, event)

	// Verify
	mockConverter.AssertExpectations(t)
	mockBus.AssertExpectations(t)
}

func TestHandler_ConversionDoneEvent_ShouldReject(t *testing.T) {
	// Setup
	mockConverter := &MockCurrencyConverter{}
	mockBus := &MockEventBus{}
	logger := slog.Default()

	handler := Handler(mockBus, mockConverter, logger)

	// Create test event - this should be rejected
	fromAmount, _ := money.New(100.0, "USD")
	toAmount, _ := money.New(85.0, "EUR")
	event := events.ConversionDoneEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     uuid.New(),
			UserID:        uuid.New(),
			CorrelationID: uuid.New(),
			FlowType:      "conversion",
		},
		FromAmount: fromAmount,
		ToAmount:   toAmount,
		RequestID:  "test-request-id",
		Timestamp:  time.Now(),
	}

	// Execute
	ctx := context.Background()
	handler(ctx, event)

	// Verify - should not call converter for ConversionDoneEvent
	mockConverter.AssertNotCalled(t, "Convert")
	mockBus.AssertNotCalled(t, "Publish")
}

func TestHandler_UnknownEvent_ShouldReject(t *testing.T) {
	// Setup
	mockConverter := &MockCurrencyConverter{}
	mockBus := &MockEventBus{}
	logger := slog.Default()

	handler := Handler(mockBus, mockConverter, logger)

	// Create an unknown event type
	amount, _ := money.New(100.0, "USD")
	event := events.DepositRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     uuid.New(),
			UserID:        uuid.New(),
			CorrelationID: uuid.New(),
			FlowType:      "deposit",
		},
		ID:        uuid.New(),
		Amount:    amount,
		Source:    "test",
		Timestamp: time.Now(),
	}

	// Execute
	ctx := context.Background()
	handler(ctx, event)

	// Verify - should not call converter for unknown event
	mockConverter.AssertNotCalled(t, "Convert")
	mockBus.AssertNotCalled(t, "Publish")
}
