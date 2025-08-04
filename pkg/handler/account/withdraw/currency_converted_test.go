package withdraw

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithdrawCurrencyConverted(t *testing.T) {
	t.Run("creates valid event", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100.0, "USD")

		// Create a WithdrawRequested event first
		withdrawRequested := events.NewWithdrawRequested(
			userID,
			accountID,
			correlationID,
			events.WithWithdrawAmount(amount),
			events.WithWithdrawID(transactionID),
		)

		// Create CurrencyConverted with the WithdrawRequested as the original request
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: *events.NewCurrencyConversionRequested(
				events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				withdrawRequested, // Pass the WithdrawRequested as the original request
				events.WithConversionAmount(amount),
				events.WithConversionTo(currency.Code("EUR")),
				events.WithConversionTransactionID(transactionID),
			),
			TransactionID:   transactionID,
			ConvertedAmount: amount,
			ConversionInfo:  nil,
		}

		event := events.NewWithdrawCurrencyConverted(cc)

		// Assert
		assert.NotNil(t, event)
		wr, ok := event.OriginalRequest.(*events.WithdrawRequested)
		require.True(t, ok, "expected WithdrawRequested")

		// Access the embedded CurrencyConverted fields directly
		assert.Equal(t, userID, wr.UserID)
		assert.Equal(t, accountID, wr.AccountID)
		assert.Equal(t, transactionID, event.TransactionID)
		assert.Equal(t, correlationID, wr.CorrelationID)
		assert.Equal(t, "withdraw", wr.FlowType)
		assert.True(t, event.ConvertedAmount.Equals(amount))
	})

	t.Run("event type is correct", func(t *testing.T) {
		event := &events.WithdrawCurrencyConverted{}
		// Type() returns the event type in the format "Withdraw.CurrencyConverted"
		assert.Equal(t, "Withdraw.CurrencyConverted", event.Type())
	})
}
