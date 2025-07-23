package conversion

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEventFactory is a mock implementation of EventFactory
type MockEventFactory struct {
	mock.Mock
}

func (m *MockEventFactory) CreateNextEvent(cre *events.ConversionRequestedEvent, convInfo *domain.ConversionInfo, convertedMoney money.Money) (domain.Event, error) {
	args := m.Called(cre, convInfo, convertedMoney)
	return args.Get(0).(domain.Event), args.Error(1)
}

func TestConversionHandler(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully converts currency and emits conversion done event", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		mockConverter := mocks.NewMockCurrencyConverter(t)
		mockFactory := &MockEventFactory{}

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := &events.ConversionRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount:        amount,
			To:            currency.EUR,
			TransactionID: transactionID,
		}

		convInfo := &domain.ConversionInfo{
			OriginalAmount:    100.0,
			OriginalCurrency:  "USD",
			ConvertedAmount:   85.0,
			ConvertedCurrency: "EUR",
			ConversionRate:    0.85,
		}

		convertedAmount, _ := money.New(85, currency.EUR)
		nextEvent := events.DepositBusinessValidationEvent{}

		// Mock expectations
		mockConverter.On("Convert", 100.0, "USD", "EUR").Return(convInfo, nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.ConversionDoneEvent")).Return(nil).Once()
		mockFactory.On("CreateNextEvent", event, convInfo, convertedAmount).Return(nextEvent, nil).Once()
		bus.On("Emit", mock.Anything, nextEvent).Return(nil).Once()

		factories := map[string]EventFactory{
			"deposit": mockFactory,
		}

		// Execute
		handler := Handler(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		mockConverter := mocks.NewMockCurrencyConverter(t)

		// Use a different event type
		event := events.DepositRequestedEvent{}

		factories := map[string]EventFactory{}

		// Execute
		handler := Handler(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err)
		// No interactions should occur with mocks
		mockConverter.AssertNotCalled(t, "Convert", mock.Anything, mock.Anything, mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles conversion error", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		mockConverter := mocks.NewMockCurrencyConverter(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := &events.ConversionRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount:        amount,
			To:            currency.EUR,
			TransactionID: transactionID,
		}

		// Mock conversion error
		mockConverter.On("Convert", 100.0, "USD", "EUR").Return((*domain.ConversionInfo)(nil), errors.New("conversion error")).Once()

		factories := map[string]EventFactory{}

		// Execute
		handler := Handler(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		assert.Error(t, err)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles unknown flow type", func(t *testing.T) {
		// Setup
		bus := mocks.NewMockBus(t)
		mockConverter := mocks.NewMockCurrencyConverter(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := &events.ConversionRequestedEvent{
			FlowEvent: events.FlowEvent{
				FlowType:      "unknown_flow", // Unknown flow type
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			Amount:        amount,
			To:            currency.EUR,
			TransactionID: transactionID,
		}

		convInfo := &domain.ConversionInfo{
			OriginalAmount:    100.0,
			OriginalCurrency:  "USD",
			ConvertedAmount:   85.0,
			ConvertedCurrency: "EUR",
			ConversionRate:    0.85,
		}

		// Mock expectations
		mockConverter.On("Convert", 100.0, "USD", "EUR").Return(convInfo, nil).Once()
		bus.On("Emit", mock.Anything, mock.AnythingOfType("events.ConversionDoneEvent")).Return(nil).Once()

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{}, // No factory for "unknown_flow"
		}

		// Execute
		handler := Handler(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		assert.NoError(t, err) // Should handle gracefully
	})
}
