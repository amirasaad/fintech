package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyConversionRequested(t *testing.T) {
	t.Run("creates_currency_conversion_requested_event_with_correct_type", func(t *testing.T) {
		amount, err := money.New(100.0, "USD")
		require.NoError(t, err)

		e := events.CurrencyConversionRequested{
			FlowEvent: events.FlowEvent{
				ID:       uuid.New(),
				FlowType: "conversion",
			},
			Amount: amount,
			To:     currency.Code("EUR"),
		}

		assert.Equal(t, "CurrencyConversion.Requested", e.Type())
	})

	t.Run("marshals_to_json_correctly", func(t *testing.T) {
		id := uuid.New()
		amount, err := money.New(100.0, "USD")
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
			To:            currency.Code("EUR"),
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

		// Check the amount field
		amountObj, ok := result["Amount"].(map[string]any)
		assert.True(t, ok, "Amount should be an object")
		// The amount is in minor units (e.g., cents)
		assert.Equal(
			t,
			"USD",
			amountObj["currency"],
			"Currency should be USD")
		assert.InEpsilon(
			t,
			100.0,
			amountObj["amount"],
			0.001,
			"Amount should be 100.00")

		// Check the to field
		assert.Equal(t, "EUR", result["To"], "To field should be EUR")

		// Check the transaction ID
		assert.Equal(
			t,
			transactionID.String(),
			result["TransactionID"],
			"TransactionID should match")
	})
}

func TestCurrencyConverted(t *testing.T) {
	t.Run("creates_currency_converted_event_with_correct_type", func(t *testing.T) {
		amount, err := money.New(
			100.0, "USD") // 1000.00 USD in minor units
		require.NoError(t, err)
		convertedAmount, err := money.New(
			850.0, "EUR") // 850.00 EUR in minor units
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
				To:            currency.Code("EUR"),
				TransactionID: uuid.New(),
			},
			TransactionID:   uuid.New(),
			ConvertedAmount: convertedAmount,
			ConversionInfo: &provider.ExchangeInfo{
				OriginalAmount:    amount.AmountFloat(),
				OriginalCurrency:  amount.Currency().String(),
				ConvertedAmount:   convertedAmount.AmountFloat(),
				ConvertedCurrency: convertedAmount.Currency().String(),
				ConversionRate:    0.85,
			},
		}

		assert.Equal(t, "CurrencyConversion.Converted", e.Type())
	})

	t.Run("marshals_to_json_correctly", func(t *testing.T) {
		// Create money amounts with minor units (e.g., cents)
		amount, err := money.New(100.0, "USD") // 1000.00 USD in minor units
		require.NoError(t, err)
		convertedAmount, err := money.New(850.0, "EUR") // 850.00 EUR in minor units
		require.NoError(t, err)

		// Create IDs
		id := uuid.New()
		userID := uuid.New()
		accountID := uuid.New()
		correlationID := uuid.New()
		transactionID := uuid.New()

		e := events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            id,
					FlowType:      "conversion",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
					Timestamp:     time.Now(),
				},
				Amount:        amount,
				To:            currency.Code("EUR"),
				TransactionID: transactionID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo: &provider.ExchangeInfo{
				OriginalAmount:    amount.AmountFloat(),
				OriginalCurrency:  amount.Currency().String(),
				ConvertedAmount:   convertedAmount.AmountFloat(),
				ConvertedCurrency: convertedAmount.Currency().String(),
				ConversionRate:    0.85,
			},
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Print the actual JSON for debugging
		t.Logf("Actual JSON: %s", string(data))

		// Verify the top-level fields
		assert.NotNil(t, result)

		// Check type field (it's set by the Type() method, not in the JSON)
		assert.Equal(t, "CurrencyConversion.Converted", e.Type())

		// Check the fields from the embedded CurrencyConversionRequested
		assert.Equal(t, id.String(), result["id"])
		assert.Equal(t, "conversion", result["flowType"])
		assert.Equal(t, userID.String(), result["userId"])
		assert.Equal(t, accountID.String(), result["accountId"])
		assert.Equal(t, correlationID.String(), result["correlationId"])

		// Check the transaction IDs (both the main one and the request one)
		assert.Equal(t, transactionID.String(), result["transactionId"])
		assert.Equal(t, transactionID.String(), result["requestTransactionId"])

		// Check the amount field
		amountObj, ok := result["amount"].(map[string]any)
		assert.True(t, ok, "amount should be an object")
		// The amount is in minor units (e.g., cents)
		// The actual value is 10000 (100.00 in minor units)
		assert.InEpsilon(
			t,
			amount.AmountFloat(),
			amountObj["amount"].(float64),
			0.001,
			"amount should be 100.00")
		assert.Equal(
			t,
			amount.Currency().String(),
			amountObj["currency"],
			"currency should be USD")

		// Check the to field
		assert.Equal(
			t,
			"EUR",
			result["to"],
			"to field should be EUR")

		// Check the converted amount
		convertedAmountObj, ok := result["convertedAmount"].(map[string]any)
		assert.True(t, ok, "convertedAmount should be an object")
		// The converted amount is in minor units (e.g., cents)
		assert.InEpsilon(
			t,
			convertedAmount.AmountFloat(),
			convertedAmountObj["amount"].(float64),
			0.001,
			"converted amount should be 850.0")
		assert.Equal(
			t,
			convertedAmount.Currency().String(),
			convertedAmountObj["currency"],
			"converted currency should be EUR")

		// Check the conversion info
		conversionInfo, ok := result["conversionInfo"].(map[string]any)
		assert.True(t, ok, "conversionInfo should be an object")
		// Check the conversion info fields (they use PascalCase in the JSON)
		// Note: The From field is 100 (not 1000) in the actual output
		assert.InEpsilon(
			t,
			100.0,
			conversionInfo["OriginalAmount"],
			0.001,
			"original amount should be 100")
		assert.Equal(
			t,
			"USD",
			conversionInfo["OriginalCurrency"],
			"original currency should be USD")
		assert.InEpsilon(
			t,
			850.0,
			conversionInfo["ConvertedAmount"],
			0.001,
			"converted amount should be 850")
		assert.Equal(
			t,
			"EUR",
			conversionInfo["ConvertedCurrency"],
			"converted currency should be EUR")
		assert.InEpsilon(
			t,
			0.85,
			conversionInfo["ConversionRate"],
			0.001,
			"conversion rate should be 0.85")

	})
}
