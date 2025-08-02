package handler_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteEventFlows(t *testing.T) {
	// Common test data
	userID := uuid.New()
	sourceAccountID := uuid.New()
	destAccountID := uuid.New()
	correlationID := uuid.New()
	amount, err := money.New(100.0, "USD")
	require.NoError(t, err)

	// Create a test event bus
	bus := eventbus.NewWithMemory(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	t.Run("deposit flow", func(t *testing.T) {
		// 1. Create DepositRequested
		depositReq := events.NewDepositRequested(
			userID,
			sourceAccountID,
			correlationID,
			events.WithDepositAmount(amount),
			events.WithDepositTransactionID(uuid.New()),
		)

		// 2. Create CurrencyConverted (simulating conversion)
		convertedAmount, _ := money.New(90.0, "EUR") // Example conversion
		convertedEvent := &events.CurrencyConverted{
			FlowEvent: events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "deposit",
				UserID:        userID,
				AccountID:     sourceAccountID,
				CorrelationID: correlationID,
			},
			TransactionID:   depositReq.TransactionID,
			ConvertedAmount: convertedAmount,
		}

		// 3. Create DepositCurrencyConverted
		depositConverted := events.NewDepositCurrencyConverted(convertedEvent, func(dcc *events.DepositCurrencyConverted) {
			dcc.DepositRequested = *depositReq
			dcc.Timestamp = time.Now()
		})

		// 4. Create DepositBusinessValidated
		businessValidated := events.NewDepositBusinessValidated(depositConverted)

		// 5. Publish and test the flow
		err := bus.Emit(context.Background(), businessValidated)
		require.NoError(t, err)

		// Verify the event structure
		dr := businessValidated.DepositRequested
		dcc := businessValidated.DepositCurrencyConverted
		assert.Equal(t, userID, dr.UserID)
		assert.Equal(t, sourceAccountID, dr.AccountID)
		assert.True(t, dcc.ConvertedAmount.Equals(convertedAmount))
	})

	t.Run("withdraw flow", func(t *testing.T) {
		// 1. Create WithdrawRequested
		withdrawReq := events.NewWithdrawRequested(
			userID,
			sourceAccountID,
			correlationID,
			events.WithWithdrawAmount(amount),
			events.WithWithdrawID(uuid.New()),
		)

		// 3. Create WithdrawCurrencyConverted
		convertedAmount, _ := money.New(90.0, "EUR") // Example conversion
		withdrawConverted := &events.WithdrawCurrencyConverted{
			WithdrawRequested: *withdrawReq,
			CurrencyConverted: events.CurrencyConverted{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     sourceAccountID,
					CorrelationID: correlationID,
				},
				TransactionID:   withdrawReq.ID,
				ConvertedAmount: convertedAmount,
			},
		}

		// 4. Create WithdrawBusinessValidated
		businessValidated := &events.WithdrawBusinessValidated{
			WithdrawCurrencyConverted: *withdrawConverted,
		}

		// 5. Publish and test the flow
		err := bus.Emit(context.Background(), businessValidated)
		require.NoError(t, err)

		// Verify the event structure
		wr := businessValidated.WithdrawRequested
		wcc := businessValidated.WithdrawCurrencyConverted
		assert.Equal(t, userID, wr.UserID)
		assert.Equal(t, sourceAccountID, wr.AccountID)
		assert.True(t, wcc.ConvertedAmount.Equals(convertedAmount))
	})

	t.Run("transfer flow", func(t *testing.T) {
		// 1. Create TransferRequsted (note the typo in "Requested")
		transferReq := &events.TransferRequested{
			FlowEvent: events.FlowEvent{
				ID:            uuid.New(),
				FlowType:      "transfer",
				UserID:        userID,
				AccountID:     sourceAccountID,
				CorrelationID: correlationID,
			},
			DestAccountID: destAccountID,
			TransactionID: uuid.New(),
			Amount:        amount,
			Timestamp:     time.Now(),
		}

		// 2. Create TransferCurrencyConverted
		convertedAmount, _ := money.New(90.0, "EUR") // Example conversion
		transferConverted := &events.TransferCurrencyConverted{
			TransferRequested: *transferReq,
			CurrencyConverted: events.CurrencyConverted{
				FlowEvent: events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "transfer",
					UserID:        userID,
					AccountID:     sourceAccountID,
					CorrelationID: correlationID,
				},
				TransactionID:   transferReq.TransactionID,
				ConvertedAmount: convertedAmount,
			},
		}

		// 3. Create TransferBusinessValidated
		businessValidated := &events.TransferBusinessValidated{
			TransferCurrencyConverted: *transferConverted,
		}

		// 4. Publish and test the flow
		err := bus.Emit(context.Background(), businessValidated)
		require.NoError(t, err)

		// Verify the event structure
		assert.Equal(t, userID, businessValidated.CurrencyConverted.UserID)
		assert.Equal(t, sourceAccountID, businessValidated.CurrencyConverted.AccountID)
		assert.Equal(t, destAccountID, businessValidated.DestAccountID)
		assert.True(t, businessValidated.ConvertedAmount.Equals(convertedAmount))
	})
}
