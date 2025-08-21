package conversion

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventFactories(t *testing.T) {
	t.Run("DepositEventFactory creates DepositCurrencyConverted", func(t *testing.T) {
		factory := &DepositEventFactory{}

		// Create a test CurrencyConverted event
		convertedAmount, _ := money.New(85.0, money.EUR)
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: uuid.New(),
			},
			TransactionID:   uuid.New(),
			ConvertedAmount: convertedAmount,
		}

		event := factory.CreateNextEvent(cc)

		require.NotNil(t, event)
		assert.IsType(t, &events.DepositCurrencyConverted{}, event)

		depositEvent, ok := event.(*events.DepositCurrencyConverted)
		require.True(t, ok)
		assert.Equal(t, cc.TransactionID, depositEvent.TransactionID)
		assert.Equal(t, cc.ConvertedAmount, depositEvent.ConvertedAmount)
	})

	t.Run("WithdrawEventFactory creates WithdrawCurrencyConverted", func(t *testing.T) {
		factory := &WithdrawEventFactory{}

		// Create a test CurrencyConverted event
		convertedAmount, _ := money.New(85.0, money.EUR)
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "withdraw",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: uuid.New(),
			},
			TransactionID:   uuid.New(),
			ConvertedAmount: convertedAmount,
		}

		event := factory.CreateNextEvent(cc)

		require.NotNil(t, event)
		assert.IsType(t, &events.WithdrawCurrencyConverted{}, event)

		withdrawEvent, ok := event.(*events.WithdrawCurrencyConverted)
		require.True(t, ok)
		assert.Equal(t, cc.TransactionID, withdrawEvent.TransactionID)
		assert.Equal(t, cc.ConvertedAmount, withdrawEvent.ConvertedAmount)
	})

	t.Run("TransferEventFactory creates TransferCurrencyConverted", func(t *testing.T) {
		factory := &TransferEventFactory{}

		// Create a test CurrencyConverted event
		convertedAmount, _ := money.New(85.0, money.EUR)
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: events.CurrencyConversionRequested{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "transfer",
					UserID:        uuid.New(),
					AccountID:     uuid.New(),
					CorrelationID: uuid.New(),
				},
				TransactionID: uuid.New(),
			},
			TransactionID:   uuid.New(),
			ConvertedAmount: convertedAmount,
		}

		event := factory.CreateNextEvent(cc)

		require.NotNil(t, event)
		assert.IsType(t, &events.TransferCurrencyConverted{}, event)

		transferEvent, ok := event.(*events.TransferCurrencyConverted)
		require.True(t, ok)
		assert.Equal(t, cc.TransactionID, transferEvent.TransactionID)
		assert.Equal(t, cc.ConvertedAmount, transferEvent.ConvertedAmount)
	})

	t.Run("EventFactory interface compliance", func(t *testing.T) {
		// Test that all factories implement the EventFactory interface
		var _ EventFactory = &DepositEventFactory{}
		var _ EventFactory = &WithdrawEventFactory{}
		var _ EventFactory = &TransferEventFactory{}
	})
}
