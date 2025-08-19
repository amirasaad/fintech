package conversion

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math"
	"reflect"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/amirasaad/fintech/pkg/registry"
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
			convInfo := &provider.ExchangeInfo{
				OriginalAmount:    100.0,
				OriginalCurrency:  money.USD.String(),
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
						events.WithConversionTo(money.EUR),
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

			// Mock registry to save the rates - both direct and inverse rates
			exchangeRateRegistryProvider.EXPECT().
				Register(
					mock.Anything,
					mock.MatchedBy(
						func(entity registry.Entity) bool {
							if entity == nil {
								return false
							}

							e := reflect.Indirect(reflect.ValueOf(entity))

							// Check for rate entities (have From, To, and Rate fields)
							fromField := e.FieldByName("From")
							toField := e.FieldByName("To")
							rateField := e.FieldByName("Rate")

							if fromField.IsValid() && toField.IsValid() && rateField.IsValid() {
								from := fromField.String()
								to := toField.String()
								rate := rateField.Float()

								// Check for direct rate (USD to EUR)
								if from == "USD" && to == "EUR" {
									return math.Abs(rate-0.85) < 0.000001
								}
								// Check for inverse rate (EUR to USD)
								if from == "EUR" && to == "USD" {
									return math.Abs(rate-1.17647) < 0.0001
								}
							}
							return false
						})).
				Return(nil).
				Twice() // Expect both direct and inverse rates

			// Mock registry for last_updated timestamp
			exchangeRateRegistryProvider.EXPECT().
				Register(
					mock.Anything,
					mock.MatchedBy(func(entity registry.Entity) bool {
						if entity == nil {
							return false
						}

						e := reflect.Indirect(reflect.ValueOf(entity))

						// Check for last_updated entity (has Timestamp field and specific ID)
						timestampField := e.FieldByName("Timestamp")
						idField := e.FieldByName("id")

						if timestampField.IsValid() && idField.IsValid() &&
							idField.String() == "exr:last_updated" {
							return true
						}
						return false
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
