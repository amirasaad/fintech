package deposit

import (
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			events.WithDepositSource("test-source"),
		)

		// 2. Create the HandleCurrencyConverted event with the same transaction ID
		convertedEvent := &events.CurrencyConverted{
			CurrencyConversionRequested: *events.NewCurrencyConversionRequested(
				depositReq.FlowEvent, depositReq),
			TransactionID:   transactionID,
			ConvertedAmount: amount,
			ConversionInfo:  nil,
		}

		// 3. Create the DepositCurrencyConverted using the factory function
		event := events.NewDepositCurrencyConverted(
			convertedEvent,
			func(dcc *events.DepositCurrencyConverted) {
				dcc.CurrencyConverted = *convertedEvent
				dcc.Timestamp = time.Now()
			},
		)

		// Assertions
		require.NotNil(t, event, "event should not be nil")

		// Verify DepositRequested fields
		t.Run("has correct DepositRequested fields", func(t *testing.T) {
			// Access fields through the embedded DepositRequested struct
			dr := event.OriginalRequest.(*events.DepositRequested)
			assert.Equal(t, userID, dr.UserID, "user ID should match")
			assert.Equal(t, accountID, dr.AccountID, "account ID should match")
			assert.Equal(t, transactionID, dr.TransactionID, "transaction ID should match")
			assert.Equal(t, correlationID, dr.CorrelationID, "correlation ID should match")
			assert.Equal(t, "deposit", dr.FlowType, "flow type should be 'deposit'")
			assert.Equal(t, "test-source", dr.Source, "source should be set")
			assert.True(t, dr.Amount.Equals(amount), "amount should match")
		})

		// Verify HandleCurrencyConverted fields
		t.Run("has correct HandleCurrencyConverted fields", func(t *testing.T) {
			cc := event.CurrencyConverted
			assert.Equal(
				t,
				transactionID,
				cc.TransactionID,
				"transaction ID should match in HandleCurrencyConverted",
			)
			assert.True(
				t,
				cc.ConvertedAmount.Equals(amount),
				"converted amount should match",
			)
			assert.Nil(
				t,
				cc.ConversionInfo,
				"conversion info should be nil",
			)
		})

		// Verify timestamps
		t.Run("has valid timestamps", func(t *testing.T) {
			assert.False(
				t,
				event.Timestamp.IsZero(),
				"timestamp should be set",
			)
		})
	})

	t.Run("has correct event type information", func(t *testing.T) {
		// Arrange
		event := &events.DepositCurrencyConverted{}

		// Act
		eventType := event.Type()

		// Assert
		assert.Equal(
			t,
			events.EventTypeDepositCurrencyConverted.String(),
			eventType,
			"type name should match the constant",
		)
	})

}
