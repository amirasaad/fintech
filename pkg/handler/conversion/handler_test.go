package conversion

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_AllFlows(t *testing.T) {
	mockConverter := mocks.NewMockCurrencyConverter(t)
	mockBus := mocks.NewMockBus(t)
	logger := slog.Default()
	ctx := context.Background()

	// Setup the factory map
	factories := map[string]EventFactory{
		"deposit":  &DepositEventFactory{},
		"withdraw": &WithdrawEventFactory{},
		"transfer": &TransferEventFactory{},
	}

	handler := Handler(mockBus, mockConverter, logger, factories)

	// Common test data
	fromAmount, _ := money.New(100.0, "USD")
	conversionInfo := &common.ConversionInfo{
		OriginalAmount:    100.0,
		OriginalCurrency:  "USD",
		ConvertedAmount:   85.0,
		ConvertedCurrency: "EUR",
		ConversionRate:    0.85,
	}

	// Test cases for each flow type
	testCases := []struct {
		name              string
		flowType          string
		expectedEventType string
	}{
		{"deposit flow", "deposit", "DepositConversionDoneEvent"},
		{"withdraw flow", "withdraw", "WithdrawConversionDoneEvent"},
		{"transfer flow", "transfer", "TransferConversionDoneEvent"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &events.ConversionRequestedEvent{
				FlowEvent: events.FlowEvent{
					FlowType:      tc.flowType,
					CorrelationID: uuid.New(),
				},
				FromAmount:    fromAmount,
				ToCurrency:    "EUR",
				TransactionID: uuid.New(),
			}

			// Mock the conversion call for this test case
			mockConverter.On("Convert", 100.0, "USD", "EUR").Return(conversionInfo, nil).Once()

			// Expect Emit to be called with the correct event type
			mockBus.On("Emit", mock.Anything, mock.MatchedBy(func(e domain.Event) bool {
				return e.Type() == tc.expectedEventType
			})).Return(nil).Once()

			// Execute
			err := handler(ctx, event)
			assert.NoError(t, err)

			// Assert that the mocks were called
			mockConverter.AssertExpectations(t)
			mockBus.AssertExpectations(t)
		})
	}
}

func TestHandler_RejectionScenarios(t *testing.T) {
	mockConverter := mocks.NewMockCurrencyConverter(t)
	mockBus := mocks.NewMockBus(t)
	logger := slog.Default()
	ctx := context.Background()

	// A minimal factory map for handler instantiation
	factories := map[string]EventFactory{"deposit": &DepositEventFactory{}}
	handler := Handler(mockBus, mockConverter, logger, factories)

	t.Run("rejects non-ConversionRequestedEvent", func(t *testing.T) {
		// This event type should be skipped by the handler
		event := events.DepositRequestedEvent{}
		err := handler(ctx, event)
		assert.NoError(t, err)
		mockConverter.AssertNotCalled(t, "Convert")
	})

	t.Run("discards event with nil TransactionID", func(t *testing.T) {
		event := &events.ConversionRequestedEvent{
			TransactionID: uuid.Nil, // Invalid
		}
		err := handler(ctx, event)
		assert.Error(t, err)
		assert.Equal(t, "invalid transaction ID", err.Error())
		mockConverter.AssertNotCalled(t, "Convert")
	})
}

// A mock EventFactory for testing error conditions
type mockEventFactory struct {
	mock.Mock
}

func (m *mockEventFactory) CreateNextEvent(
	cre *events.ConversionRequestedEvent,
	convInfo *common.ConversionInfo,
	convertedMoney money.Money,
) (domain.Event, error) {
	args := m.Called(cre, convInfo, convertedMoney)
	// Ensure we don't return a nil event, which would cause a panic
	if args.Get(0) == nil {
		return &events.ConversionDoneEvent{}, args.Error(1)
	}
	return args.Get(0).(domain.Event), args.Error(1)
}

func TestHandler_ErrorAndEdgeCases(t *testing.T) {
	mockConverter := mocks.NewMockCurrencyConverter(t)
	logger := slog.Default()
	ctx := context.Background()
	validID := uuid.New()
	validMoney, _ := money.New(100.0, "USD")
	convInfo := &common.ConversionInfo{
		OriginalAmount: 100.0, OriginalCurrency: "USD",
		ConvertedAmount: 85.0, ConvertedCurrency: "EUR", ConversionRate: 0.85,
	}

	t.Run("error from converter.Convert", func(t *testing.T) {
		factories := map[string]EventFactory{"deposit": &DepositEventFactory{}}
		handler := Handler(mocks.NewMockBus(t), mockConverter, logger, factories)
		event := &events.ConversionRequestedEvent{
			FlowEvent:  events.FlowEvent{FlowType: "deposit"},
			FromAmount: validMoney, ToCurrency: "EUR", TransactionID: validID,
		}

		mockConverter.On("Convert", 100.0, "USD", "EUR").Return(nil, errors.New("converter error")).Once()
		err := handler(ctx, event)
		assert.Error(t, err)
		mockConverter.AssertExpectations(t)
	})

	t.Run("error from money.New for converted", func(t *testing.T) {
		factories := map[string]EventFactory{"deposit": &DepositEventFactory{}}
		handler := Handler(mocks.NewMockBus(t), mockConverter, logger, factories)
		event := &events.ConversionRequestedEvent{
			FlowEvent:  events.FlowEvent{FlowType: "deposit"},
			FromAmount: validMoney, ToCurrency: "INVALID", TransactionID: validID,
		}
		badConvInfo := &common.ConversionInfo{ConvertedCurrency: "INVALID"}
		mockConverter.On("Convert", 100.0, "USD", "INVALID").Return(badConvInfo, nil).Once()
		err := handler(ctx, event)
		assert.Error(t, err)
	})

	t.Run("unknown flow type gracefully ignored", func(t *testing.T) {
		factories := map[string]EventFactory{"deposit": &DepositEventFactory{}}
		handler := Handler(mocks.NewMockBus(t), mockConverter, logger, factories)
		event := &events.ConversionRequestedEvent{
			FlowEvent:  events.FlowEvent{FlowType: "unknown"},
			FromAmount: validMoney, ToCurrency: "EUR", TransactionID: validID,
		}

		mockConverter.On("Convert", 100.0, "USD", "EUR").Return(convInfo, nil).Once()
		err := handler(ctx, event)
		assert.NoError(t, err, "handler should not return an error for an unknown flow type")
	})

	t.Run("error from event factory", func(t *testing.T) {
		mockF := &mockEventFactory{}
		factories := map[string]EventFactory{"deposit": mockF}
		handler := Handler(mocks.NewMockBus(t), mockConverter, logger, factories)
		event := &events.ConversionRequestedEvent{
			FlowEvent:  events.FlowEvent{FlowType: "deposit"},
			FromAmount: validMoney, ToCurrency: "EUR", TransactionID: validID,
		}

		convertedMoney, _ := money.New(85.0, "EUR")
		mockConverter.On("Convert", 100.0, "USD", "EUR").Return(convInfo, nil).Once()
		mockF.On("CreateNextEvent", event, convInfo, convertedMoney).Return(nil, errors.New("factory error")).Once()

		err := handler(ctx, event)
		assert.Error(t, err)
	})

	t.Run("error from bus.Emit", func(t *testing.T) {
		busWithErr := mocks.NewMockBus(t)
		factories := map[string]EventFactory{"deposit": &DepositEventFactory{}}
		handler := Handler(busWithErr, mockConverter, logger, factories)
		event := &events.ConversionRequestedEvent{
			FlowEvent:  events.FlowEvent{FlowType: "deposit"},
			FromAmount: validMoney, ToCurrency: "EUR", TransactionID: validID,
		}

		mockConverter.On("Convert", 100.0, "USD", "EUR").Return(convInfo, nil).Once()
		busWithErr.On("Emit", mock.Anything, mock.AnythingOfType("events.DepositConversionDoneEvent")).Return(errors.New("bus error")).Once()

		err := handler(ctx, event)
		assert.Error(t, err)
	})
}
