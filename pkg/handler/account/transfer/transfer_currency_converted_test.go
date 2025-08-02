package transfer

import (
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTransferCurrencyConverted(t *testing.T) {
	t.Run("creates valid event", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		accountID := uuid.New()
		destAccountID := uuid.New()
		transactionID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100.0, "USD")

		event := events.NewTransferCurrencyConverted(
			&events.TransferRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "transfer",
					UserID:        userID,
					AccountID:     accountID,
					CorrelationID: correlationID,
				},
				DestAccountID: destAccountID,
				TransactionID: transactionID,
				Timestamp:     time.Now(),
			},
			func(tr *events.TransferCurrencyConverted) {
				tr.ConvertedAmount = amount
			},
		)

		// Assert
		assert.NotNil(t, event)
		assert.Equal(t, userID, event.TransferRequested.UserID)
		assert.Equal(t, accountID, event.TransferRequested.AccountID)
		assert.Equal(t, transactionID, event.TransferRequested.TransactionID)
		assert.Equal(t, correlationID, event.TransferRequested.CorrelationID)
		assert.Equal(t, "transfer", event.TransferRequested.FlowType)
		assert.True(t, event.ConvertedAmount.Equals(amount))
	})

	t.Run("event type is correct", func(t *testing.T) {
		event := &events.TransferCurrencyConverted{}
		expectedType := "TransferCurrencyConverted"
		assert.Equal(t, expectedType, event.Type())
	})
}
