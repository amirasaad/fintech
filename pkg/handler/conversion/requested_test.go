package conversion

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEventFactory is a mock implementation of EventFactory
type MockEventFactory struct {
	mock.Mock
}

// CreateNextEvent is a mock implementation of EventFactory.CreateNextEvent
func (m *MockEventFactory) CreateNextEvent(cr *events.CurrencyConverted) events.Event {
	args := m.Called(cr)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(events.Event)
}

func TestConversionHandler(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	t.Run(
		"successfully converts currency and emits conversion done event",
		func(
			t *testing.T,
		) {
			// Setup
			bus := mocks.NewBus(t)
			mockConverter := mocks.NewCurrencyConverter(t)
			mockFactory := &MockEventFactory{}

			userID := uuid.New()
			accountID := uuid.New()
			transactionID := uuid.New()
			amount, _ := money.New(100, currency.USD)

			event := &events.CurrencyConversionRequested{
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

			correlationID := uuid.New()
			nextEvent := events.NewDepositCurrencyConverted(
				events.NewCurrencyConverted(
					events.NewCurrencyConversionRequested(
						events.FlowEvent{CorrelationID: correlationID},
						nil,
						events.WithConversionAmount(amount),
						events.WithConversionTo(currency.EUR),
						events.WithConversionTransactionID(transactionID),
					),
				),
			)

			// Mock expectations - use currency.Code types for currency codes
			mockConverter.EXPECT().
				Convert(100.0, currency.USD, currency.EUR).
				Return(convInfo, nil).
				Once()

			bus.EXPECT().
				Emit(mock.Anything, mock.MatchedBy(func(e events.Event) bool {
					_, ok := e.(*events.CurrencyConverted)
					return ok
				})).
				Return(nil).
				Once()

			mockFactory.On(
				"CreateNextEvent",
				mock.MatchedBy(func(e *events.CurrencyConverted) bool {
					return e != nil
				}),
			).Return(nextEvent).Once()

			bus.EXPECT().
				Emit(mock.Anything, mock.MatchedBy(func(e events.Event) bool {
					_, ok := e.(*events.DepositCurrencyConverted)
					return ok
				})).
				Return(nil).
				Once()

			factories := map[string]EventFactory{
				"deposit": mockFactory,
			}

			// Execute
			handler := HandleRequested(bus, mockConverter, logger, factories)
			err := handler(ctx, event)

			// Assert
			assert.NoError(t, err)
		})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewBus(t)
		mockConverter := mocks.NewCurrencyConverter(t)

		// Use a different event type
		event := events.DepositRequested{}

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{},
		}

		// Execute
		handler := HandleRequested(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
		// No interactions should occur with mocks
		mockConverter.AssertNotCalled(t, "Convert", mock.Anything, mock.Anything, mock.Anything)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles conversion error", func(t *testing.T) {
		// Setup
		bus := mocks.NewBus(t)
		mockConverter := mocks.NewCurrencyConverter(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewCurrencyConversionRequested(
			events.FlowEvent{
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			nil,
			func(ccr *events.CurrencyConversionRequested) {
				ccr.Amount = amount
				ccr.To = currency.EUR
				ccr.TransactionID = transactionID
			},
		)

		// Mock conversion error
		mockConverter.EXPECT().
			Convert(100.0, currency.USD, currency.EUR).
			Return(nil, errors.New("conversion error")).
			Once()

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{},
		}

		// Execute
		handler := HandleRequested(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		bus.AssertNotCalled(
			t,
			"Emit",
			mock.Anything,
			mock.AnythingOfType("*events.ConversionDoneEvent"),
		)
		bus.AssertNotCalled(
			t,
			"Emit",
			mock.Anything,
			mock.AnythingOfType("*events.DepositBusinessValidationEvent"),
		)
	})

	t.Run("handles unknown flow type", func(t *testing.T) {
		// Setup
		bus := mocks.NewBus(t)
		mockConverter := mocks.NewCurrencyConverter(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, currency.USD)

		event := events.NewCurrencyConversionRequested(
			events.FlowEvent{
				FlowType:      "unknown_flow", // Unknown flow type
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: uuid.New(),
			},
			nil,
			func(ccr *events.CurrencyConversionRequested) {
				ccr.Amount = amount
				ccr.To = currency.EUR
				ccr.TransactionID = transactionID
			},
		)

		// No mock expectations for unknown flow type

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{}, // No factory for "unknown_flow"
		}

		// Execute
		handler := HandleRequested(bus, mockConverter, logger, factories)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown flow type")
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
