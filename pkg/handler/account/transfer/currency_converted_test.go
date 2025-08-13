package transfer

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTransferCurrencyConverted(t *testing.T) {
	t.Run("creates valid event", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100.0, "USD")

		event := events.NewTransferCurrencyConverted(
			&events.CurrencyConverted{
				CurrencyConversionRequested: *events.NewCurrencyConversionRequested(
					events.FlowEvent{
						ID:            uuid.New(),
						FlowType:      "transfer",
						UserID:        userID,
						AccountID:     accountID,
						CorrelationID: correlationID,
					},
					nil,
					events.WithConversionAmount(amount),
					events.WithConversionTo(currency.Code("EUR")),
					events.WithConversionTransactionID(transactionID),
				),
				TransactionID:   transactionID,
				ConvertedAmount: amount,
				ConversionInfo:  nil,
			},
		)

		// Assert
		assert.NotNil(t, event)
		assert.Equal(t, userID, event.UserID)
		assert.Equal(t, accountID, event.AccountID)
		assert.Equal(t, transactionID, event.TransactionID)
		assert.Equal(t, correlationID, event.CorrelationID)
		assert.Equal(t, "transfer", event.FlowType)
		assert.True(t, event.ConvertedAmount.Equals(amount))
	})

	t.Run("event type is correct", func(t *testing.T) {
		event := &events.TransferCurrencyConverted{}
		expectedType := events.EventTypeTransferCurrencyConverted.String()
		assert.Equal(t, expectedType, event.Type())
	})
}
