package conversion

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
	"github.com/amirasaad/fintech/pkg/service/exchange"
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
			exchangeRateProvider := mocks.NewExchangeRateProvider(t)
			exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)
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
			convInfo := &provider.ExchangeInfo{
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
			exchangeRateRegistryProvider.EXPECT().
				Get(mock.Anything, "USD:EUR").
				Return(nil, provider.ErrExchangeRateUnavailable).
				Once()

			// Mock provider name and health check (called multiple times in the provider loop)
			exchangeRateProvider.EXPECT().
				Name().
				Return("mock-provider").
				Maybe()

			exchangeRateProvider.EXPECT().
				IsHealthy().
				Return(true).
				Maybe()

			// Mock provider to return exchange rate
			exchangeRateProvider.EXPECT().
				GetRate(mock.Anything, "USD", "EUR").
				Return(convInfo, nil).
				Once()

			// Mock registry to save the direct rate (inverse rates are no longer cached)
			exchangeRateRegistryProvider.EXPECT().
				Register(mock.Anything, mock.MatchedBy(func(entity registry.Entity) bool {
					er, ok := entity.(*exchange.ExchangeRateInfo)
					return ok && er.From == "USD" && er.To == "EUR" && er.Rate == 0.85
				})).
				Return(nil).
				Once()

			// Expect CurrencyConverted event
			bus.EXPECT().
				Emit(mock.Anything, mock.AnythingOfType("*events.CurrencyConverted")).
				Return(nil).
				Once()

			// Expect DepositCurrencyConverted event
			bus.EXPECT().
				Emit(mock.Anything, mock.AnythingOfType("*events.DepositCurrencyConverted")).
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
		exchangeRateProvider := mocks.NewExchangeRateProvider(t)
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
		exchangeRateProvider := mocks.NewExchangeRateProvider(t)
		exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)

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

		// Mock registry to return rate not found (we no longer check for inverse rates)
		exchangeRateRegistryProvider.EXPECT().
			Get(mock.Anything, "USD:EUR").
			Return(nil, provider.ErrExchangeRateUnavailable).
			Once()

		// Mock provider name and health check
		exchangeRateProvider.EXPECT().
			Name().
			Return("mock-provider").
			Maybe()

		exchangeRateProvider.EXPECT().
			IsHealthy().
			Return(true).
			Maybe()

		// Mock provider to return error
		exchangeRateProvider.EXPECT().
			GetRate(mock.Anything, "USD", "EUR").
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
		exchangeRateProvider := mocks.NewExchangeRateProvider(t)
		exchangeRateRegistryProvider := mocks.NewRegistryProvider(t)

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
