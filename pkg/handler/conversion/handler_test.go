package conversion

import (
	"context"
	"fmt"
	"testing"
	"time"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/eventbus"
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
	handlers map[string][]eventbus.HandlerFunc
}

func (m *MockEventBus) Emit(ctx context.Context, event domain.Event) error {
	handlers := m.handlers[event.Type()]
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockEventBus) Register(eventType string, handler eventbus.HandlerFunc) {
	if m.handlers == nil {
		m.handlers = make(map[string][]eventbus.HandlerFunc)
	}
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

func TestHandler_ConversionRequestedEvent(t *testing.T) {
	// Setup
	mockConverter := &MockCurrencyConverter{}
	mockBus := &MockEventBus{}
	logger := slog.Default()

	handler := Handler(mockBus, mockConverter, logger)

	// Create test event
	fromAmount, _ := money.New(100.0, "USD")
	event := &events.ConversionRequestedEvent{
		FlowEvent: events.FlowEvent{
			AccountID:     uuid.New(),
			UserID:        uuid.New(),
			CorrelationID: uuid.New(),
			FlowType:      "conversion",
		},
		FromAmount:    fromAmount,
		ToCurrency:    "EUR",
		RequestID:     "test-request-id",
		TransactionID: uuid.New(), // Ensure TransactionID is set
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
	mockBus.Register("ConversionDoneEvent", func(ctx context.Context, event domain.Event) error {
		doneEvent, ok := event.(*events.ConversionDoneEvent)
		if !ok {
			return fmt.Errorf("unexpected event type: %T", event)
		}
		if doneEvent.FromAmount.Amount() != 100.0 || string(doneEvent.FromAmount.Currency()) != "USD" || doneEvent.ToAmount.Amount() != 85.0 || string(doneEvent.ToAmount.Currency()) != "EUR" {
			return fmt.Errorf("unexpected conversion result: %+v", doneEvent)
		}
		return nil
	})

	// Execute
	ctx := context.Background()
	handler(ctx, event) //nolint:errcheck

	// Verify
	mockConverter.AssertExpectations(t)
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
	handler(ctx, event) //nolint:errcheck

	// Verify - should not call converter for ConversionDoneEvent
	mockConverter.AssertNotCalled(t, "Convert")
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
	handler(ctx, event) //nolint:errcheck

	// Verify - should not call converter for unknown event
	mockConverter.AssertNotCalled(t, "Convert")
}
