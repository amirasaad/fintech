package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider/exchange"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyConversionRequested(t *testing.T) {
	t.Run("creates_currency_conversion_requested_event_with_correct_type", func(t *testing.T) {
		amount, err := money.New(100.0, money.USD)
		require.NoError(t, err)

		e := events.CurrencyConversionRequested{
			FlowEvent: events.FlowEvent{
				ID:       uuid.New(),
				FlowType: "conversion",
			},
			Amount: amount,
			To:     money.EUR,
		}

		assert.Equal(t, "CurrencyConversion.Requested", e.Type())
	})

	t.Run("marshals_to_json_correctly", func(t *testing.T) {
		id := uuid.New()
		amount, err := money.New(100.0, money.USD)
		require.NoError(t, err)

		// Set up test data
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		transactionID := uuid.New()
		timestamp := time.Now()

		e := events.CurrencyConversionRequested{
			FlowEvent: events.FlowEvent{
				ID:            id,
				FlowType:      "conversion",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
				Timestamp:     timestamp,
			},
			Amount:        amount,
			To:            money.EUR,
			TransactionID: transactionID,
		}

		// Marshal to JSON
		data, err := json.Marshal(e)
		require.NoError(t, err)

		// Print the actual JSON for debugging
		t.Logf("Actual JSON: %s", data)

		// Unmarshal back to a map for inspection
		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Check the type field (it's set by the Type() method, not in the JSON)
		assert.Equal(t, "CurrencyConversion.Requested", e.Type())

		// The JSON uses the exact field names from the struct (PascalCase for exported fields)
		// Check the fields from the embedded FlowEvent
		assert.Equal(t, id.String(), result["ID"])
		assert.Equal(t, "conversion", result["FlowType"])
		assert.Equal(t, userID.String(), result["UserID"])
		assert.Equal(t, accountID.String(), result["AccountID"])
		assert.Equal(t, correlationID.String(), result["CorrelationID"])

		// Parse and check timestamp
		ts, err := time.Parse(time.RFC3339Nano, result["Timestamp"].(string))
		require.NoError(t, err)
		assert.WithinDuration(
			t,
			timestamp,
			ts,
			time.Second,
			"Timestamp should be approximately equal",
		)

		// Check the direct fields of CurrencyConversionRequested
		amountMap, ok := result["Amount"].(map[string]any)
		require.True(t, ok, "Amount should be an object")
		assert.InEpsilon(
			t,
			float64(amount.Amount()),
			amountMap["amount"].(float64),
			0.0001,
			"Amount should match",
		)
		assert.Equal(
			t,
			"USD",
			amountMap["currency"],
			"Currency should be USD",
		)

		// Check the to field
		assert.Equal(
			t,
			"EUR",
			result["To"],
			"To field should be EUR",
		)

		// Check the transaction ID
		assert.Equal(
			t,
			transactionID.String(),
			result["TransactionID"],
			"TransactionID should match",
		)
	})
}

func TestCurrencyConverted(t *testing.T) {
	t.Run("creates_currency_converted_event_with_correct_type", func(t *testing.T) {
		amount, err := money.New(100.0, money.USD) // 100.00 USD in cents
		require.NoError(t, err)
		convertedAmount, err := money.New(85.0, money.EUR) // 85.00 EUR in cents
		require.NoError(t, err)

		e := events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "conversion",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
					Timestamp:     time.Now(),
				},
				Amount:        amount,
				To:            money.EUR,
				TransactionID: uuid.New(),
			},
			TransactionID:   uuid.New(),
			ConvertedAmount: convertedAmount,
			ConversionInfo: &exchange.RateInfo{
				FromCurrency: amount.Currency().String(),
				ToCurrency:   convertedAmount.Currency().String(),
				Rate:         0.85,
			},
		}

		assert.Equal(t, "CurrencyConversion.Converted", e.Type())
	})

	t.Run("marshals_to_json_correctly", func(t *testing.T) {
		// Set up test data
		id := uuid.New()
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		transactionID := uuid.New()
		timestamp := time.Now()

		// Create money amounts using the money package
		amount, err := money.New(100.0, money.USD) // 100.00 USD in cents (10000)
		require.NoError(t, err)
		convertedAmount, err := money.New(85.0, money.EUR) // 85.00 EUR in cents (8500)
		require.NoError(t, err)

		e := events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            id,
					FlowType:      "conversion",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
					Timestamp:     timestamp,
				},
				Amount:        amount,
				To:            money.EUR,
				TransactionID: transactionID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo: &exchange.RateInfo{
				FromCurrency: amount.Currency().String(),
				ToCurrency:   convertedAmount.Currency().String(),
				Rate:         0.85,
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(e)
		require.NoError(t, err)

		// Print the actual JSON for debugging
		t.Logf("Actual JSON: %s", data)

		// Unmarshal back to a map for inspection
		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Check the type field (it's set by the Type() method, not in the JSON)
		assert.Equal(t, "CurrencyConversion.Converted", e.Type())

		// The JSON uses the exact field names from the struct (PascalCase for exported fields)
		// Check the fields from the embedded FlowEvent
		assert.Equal(t, id.String(), result["id"])
		assert.Equal(t, "conversion", result["flowType"])
		assert.Equal(t, userID.String(), result["userId"])
		assert.Equal(t, accountID.String(), result["accountId"])
		assert.Equal(t, correlationID.String(), result["correlationId"])
		ts, err := time.Parse(time.RFC3339Nano, result["timestamp"].(string))
		require.NoError(t, err)
		assert.WithinDuration(
			t,
			timestamp,
			ts,
			time.Second,
			"Timestamp should be approximately equal",
		)

		// Check the direct fields of CurrencyConversionRequested
		amountMap, ok := result["amount"].(map[string]any)
		require.True(t, ok, "amount should be an object")
		assert.InEpsilon(
			t,
			float64(10000),
			amountMap["amount"].(float64),
			0.0001,
			"Amount should be 10000 (100.00 USD in cents)",
		)
		assert.Equal(
			t,
			"USD",
			amountMap["currency"],
			"Currency should be USD",
		)

		// Check the to field
		assert.Equal(
			t,
			"EUR",
			result["to"],
			"To field should be EUR",
		)

		// Check the transaction ID
		assert.Equal(
			t,
			transactionID.String(),
			result["transactionId"],
			"TransactionID should match",
		)

		// Check the converted amount
		convertedAmountMap, ok := result["convertedAmount"].(map[string]any)
		require.True(t, ok, "convertedAmount should be an object")
		assert.InEpsilon(
			t,
			float64(8500),
			convertedAmountMap["amount"].(float64),
			0.0001,
			"Converted amount should be 8500 (85.00 EUR in cents)",
		)
		assert.Equal(
			t,
			"EUR",
			convertedAmountMap["currency"],
			"Converted amount currency should be EUR",
		)

		// Check the conversion info
		conversionInfo, ok := result["conversionInfo"].(map[string]any)
		require.True(t, ok, "conversionInfo should be an object")

		// Check the rate info fields
		assert.Equal(t, "USD", conversionInfo["from_currency"],
			"From currency should be USD")
		assert.Equal(t, "EUR", conversionInfo["to_currency"],
			"To currency should be EUR")
		assert.InEpsilon(t, 0.85, conversionInfo["rate"], 0.001,
			"Conversion rate should be 0.85")

		// Re-declare variables to avoid shadowing
		amountMap = result["amount"].(map[string]any)
		convertedAmountMap = result["convertedAmount"].(map[string]any)

		assert.InEpsilon(t, 100.0, amountMap["amount"].(float64)/100, 0.001,
			"Original amount should be 100.0")
		assert.Equal(t, "USD", amountMap["currency"],
			"Original currency should be USD")

		assert.InEpsilon(t, 85.0, convertedAmountMap["amount"].(float64)/100, 0.001,
			"Converted amount should be 85.0")
		assert.Equal(t, "EUR", convertedAmountMap["currency"],
			"Converted currency should be EUR")
	})
}
