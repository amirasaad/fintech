package deposit

import (
	"reflect"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WithDepositSource is a test helper to set the source on a DepositRequested event
func WithDepositSource(source string) events.DepositRequestedOpt {
	return func(e *events.DepositRequested) {
		e.Source = source
	}
}

func TestDepositCurrencyConverted(t *testing.T) {
	t.Run("creates valid event with all required fields", func(t *testing.T) {
		// Setup test data
		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		correlationID := uuid.New()
		amount, err := money.New(100.0, "USD")
		require.NoError(t, err, "should create money amount without error")

		// 1. Create the base DepositRequested event with all required fields
		depositReq := events.NewDepositRequested(
			userID,
			accountID,
			correlationID,
			events.WithDepositAmount(amount),
			events.WithDepositTransactionID(transactionID),
			events.WithDepositID(uuid.New()),
			WithDepositSource("test-source"),
		)

		// 2. Create the CurrencyConverted event with the same transaction ID
		convertedEvent := &events.CurrencyConverted{
			FlowEvent: events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
				Timestamp:     time.Now(),
			},
			TransactionID:   transactionID,
			ConvertedAmount: amount,
			ConversionInfo:  nil,
		}

		// 3. Create the DepositCurrencyConverted using the factory function
		event := events.NewDepositCurrencyConverted(convertedEvent, func(dcc *events.DepositCurrencyConverted) {
			dcc.DepositRequested = *depositReq
			dcc.Timestamp = time.Now()
		})

		// Assertions
		require.NotNil(t, event, "event should not be nil")

		// Verify DepositRequested fields
		t.Run("has correct DepositRequested fields", func(t *testing.T) {
			// Access fields through the embedded DepositRequested struct
			dr := event.DepositRequested
			assert.Equal(t, userID, dr.UserID, "user ID should match")
			assert.Equal(t, accountID, dr.AccountID, "account ID should match")
			assert.Equal(t, transactionID, dr.TransactionID, "transaction ID should match")
			assert.Equal(t, correlationID, dr.CorrelationID, "correlation ID should match")
			assert.Equal(t, "deposit", dr.FlowType, "flow type should be 'deposit'")
			assert.Equal(t, "test-source", dr.Source, "source should be set")
			assert.True(t, dr.Amount.Equals(amount), "amount should match")
		})

		// Verify CurrencyConverted fields
		t.Run("has correct CurrencyConverted fields", func(t *testing.T) {
			cc := event.CurrencyConverted
			assert.Equal(t, transactionID, cc.TransactionID, "transaction ID should match in CurrencyConverted")
			assert.True(t, cc.ConvertedAmount.Equals(amount), "converted amount should match")
			assert.Nil(t, cc.ConversionInfo, "conversion info should be nil")
		})

		// Verify timestamps
		t.Run("has valid timestamps", func(t *testing.T) {
			assert.False(t, event.Timestamp.IsZero(), "timestamp should be set")
			assert.False(t, event.DepositRequested.Timestamp.IsZero(), "deposit requested timestamp should be set")
		})
	})

	t.Run("has correct event type information", func(t *testing.T) {
		// Arrange
		event := &events.DepositCurrencyConverted{}

		// Act
		eventType := event.Type()

		// Assert
		assert.Equal(t, "DepositCurrencyConverted", eventType, "type name should be correct")

		// Verify the type can be used for registration
		// The event is already of type *events.DepositCurrencyConverted, no need to assert
	})

	t.Run("can be used with event bus registration", func(t *testing.T) {
		// This test verifies the event can be used with reflect.Type for registration
		eventType := reflect.TypeOf((*events.DepositCurrencyConverted)(nil)).Elem()
		assert.Equal(t, "DepositCurrencyConverted", eventType.Name(), "should get correct type name")

		// Verify we can create a new instance of the event
		eventValue := reflect.New(eventType).Interface()
		_, ok := eventValue.(*events.DepositCurrencyConverted)
		assert.True(t, ok, "should be able to create new instance of DepositCurrencyConverted")
	})
}
