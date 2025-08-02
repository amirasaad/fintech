package withdraw

import (
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWithdrawCurrencyConverted(t *testing.T) {
	t.Run("creates valid event", func(t *testing.T) {
		// Setup
		userID := uuid.New()
		accountID := uuid.New()
		transactionID := uuid.New()
		correlationID := uuid.New()
		amount, _ := money.New(100.0, "USD")
		event := events.NewWithdrawCurrencyConverted(&events.WithdrawRequested{
			FlowEvent: events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "withdraw",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			ID:        uuid.New(),
			Amount:    amount,
			Timestamp: time.Now(),
		}, &events.CurrencyConverted{
			FlowEvent: events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "withdraw",
				UserID:        userID,
				AccountID:     accountID,
				CorrelationID: correlationID,
			},
			TransactionID:   transactionID,
			ConvertedAmount: amount,
			ConversionInfo:  nil,
		})

		// Assert
		assert.NotNil(t, event)
		wr := event.WithdrawRequested
		cc := event.CurrencyConverted
		assert.Equal(t, userID, wr.UserID)
		assert.Equal(t, accountID, wr.AccountID)
		assert.Equal(t, transactionID, cc.TransactionID)
		assert.Equal(t, correlationID, wr.CorrelationID)
		assert.Equal(t, "withdraw", wr.FlowType)
		assert.True(t, event.ConvertedAmount.Equals(amount))
	})

	t.Run("event type is correct", func(t *testing.T) {
		event := &events.WithdrawCurrencyConverted{}
		// Type() now returns the event type name directly
		assert.Equal(t, "WithdrawCurrencyConverted", event.Type())
	})
}
