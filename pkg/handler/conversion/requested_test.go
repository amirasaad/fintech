package conversion

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
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
			exchangeRateProvider := mocks.NewExchangeProvider(t)
			exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)
			mockFactory := &MockEventFactory{}

			userID := uuid.New()
			accountID := uuid.New()
			transactionID := uuid.New()
			amount, _ := money.New(100, money.USD)

			event := &events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: uuid.New(),
				},
				Amount:        amount,
				To:            money.EUR,
				TransactionID: transactionID,
			}
			convInfo := &exchange.RateInfo{
				FromCurrency: money.USD.String(),
				ToCurrency:   "EUR",
				Rate:         0.85,
			}

			correlationID := uuid.New()
			nextEvent := events.NewDepositCurrencyConverted(
				events.NewCurrencyConverted(
					events.NewCurrencyConversionRequested(
						events.FlowEvent{CorrelationID: correlationID},
						nil,
						events.WithConversionAmount(amount),
						events.WithConversionTo(money.EUR),
						events.WithConversionTransactionID(transactionID),
					),
				),
			)

			// Setup common mocks that can be called multiple times
			exchangeRateProvider.EXPECT().
				Metadata().
				Return(exchange.ProviderMetadata{Name: "test-provider", IsActive: true}).
				Maybe()

			exchangeRateProvider.EXPECT().
				CheckHealth(mock.Anything).
				Return(nil).
				Maybe()

			// Mock registry - first cache miss, then register two rates
			exchangeRateRegistryProvider.EXPECT().
				Get(mock.Anything, "USD:EUR").
				Return(nil, exchange.ErrProviderUnavailable).
				Once()

			// Mock exchange rate fetch
			exchangeRateProvider.EXPECT().
				FetchRate(mock.Anything, "USD", "EUR").
				Return(convInfo, nil).
				Once()

			// Mock rate registration - accept any rate registration
			exchangeRateRegistryProvider.EXPECT().
				Register(mock.Anything, mock.MatchedBy(func(e interface{}) bool {
					// Accept any non-nil entity
					return e != nil
				})).
				Return(nil).
				Times(2) // Expect two registrations (direct and inverse rates)

			// Expect event emissions - use more flexible matching
			bus.EXPECT().
				Emit(mock.Anything, mock.MatchedBy(func(e interface{}) bool {
					_, ok := e.(*events.CurrencyConverted)
					return ok
				})).
				Return(nil).
				Once()

			bus.EXPECT().
				Emit(mock.Anything, mock.MatchedBy(func(e interface{}) bool {
					_, ok := e.(*events.DepositCurrencyConverted)
					return ok
				})).
				Return(nil).
				Once()

			// Setup mock for next event creation - accept any CurrencyConverted event
			mockFactory.On(
				"CreateNextEvent",
				mock.AnythingOfType("*events.CurrencyConverted"),
			).Return(nextEvent).Once()

			factories := map[string]EventFactory{
				"deposit": mockFactory,
			}

			// Execute
			handler := HandleRequested(
				bus,
				exchangeRateRegistryProvider,
				exchangeRateProvider,
				logger,
				factories,
			)
			err := handler(ctx, event)

			// Assert
			assert.NoError(t, err)
		})

	t.Run("handles unexpected event type gracefully", func(t *testing.T) {
		// Setup
		bus := mocks.NewBus(t)
		exchangeRateProvider := mocks.NewExchangeProvider(t)
		exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)

		// Use a different event type
		event := events.DepositRequested{}

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{},
		}

		// Execute
		handler := HandleRequested(
			bus,
			exchangeRateRegistryProvider,
			exchangeRateProvider,
			logger,
			factories,
		)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected event type")
		// No interactions should occur with mocks

		exchangeRateRegistryProvider.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
		exchangeRateProvider.AssertNotCalled(
			t,
			"GetRate",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		)
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})

	t.Run("handles conversion error", func(t *testing.T) {
		// Setup
		bus := mocks.NewBus(t)
		exchangeRateProvider := mocks.NewExchangeProvider(t)
		exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, money.USD)

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
				ccr.To = money.EUR
				ccr.TransactionID = transactionID
			},
		)

		// Mock registry to return rate not found (we no longer check for inverse rates)
		exchangeRateRegistryProvider.EXPECT().
			Get(mock.Anything, "USD:EUR").
			Return(nil, exchange.ErrProviderUnavailable).
			Once()

		exchangeRateProvider.EXPECT().
			CheckHealth(mock.Anything).
			Return(nil).
			Maybe()

		// Mock provider to return error
		exchangeRateProvider.EXPECT().
			FetchRate(mock.Anything, "USD", "EUR").
			Return(nil, errors.New("conversion error")).
			Once()

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{},
		}

		// Execute
		handler := HandleRequested(
			bus,
			exchangeRateRegistryProvider,
			exchangeRateProvider,
			logger,
			factories,
		)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		bus.AssertNotCalled(
			t,
			"Emit",
			mock.Anything,
			mock.AnythingOfType("*events.CurrencyConverted"),
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
		exchangeRateProvider := mocks.NewExchangeProvider(t)
		exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)

		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		amount, _ := money.New(100, money.USD)

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
				ccr.To = money.EUR
				ccr.TransactionID = transactionID
			},
		)

		// No mock expectations for unknown flow type

		factories := map[string]EventFactory{
			"deposit": &MockEventFactory{}, // No factory for "unknown_flow"
		}

		// Execute
		handler := HandleRequested(
			bus,
			exchangeRateRegistryProvider,
			exchangeRateProvider,
			logger,
			factories,
		)
		err := handler(ctx, event)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown flow type")
		bus.AssertNotCalled(t, "Emit", mock.Anything, mock.Anything)
	})
}
