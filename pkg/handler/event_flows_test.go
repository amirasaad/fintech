package handler_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/amirasaad/fintech/infra/eventbus"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
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
	amount, err := money.New(100.0, money.USD)
	require.NoError(t, err)

	// Create a test event bus
	bus := eventbus.NewWithMemory(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	t.Run("deposit flow", func(t *testing.T) {
		// 1. Create DepositRequested
		dr := events.NewDepositRequested(
			userID,
			sourceAccountID,
			correlationID,
			events.WithDepositAmount(amount),
			events.WithDepositTransactionID(uuid.New()),
		)

		// 2. Create CurrencyConverted (simulating conversion)
		convertedAmount, _ := money.New(90.0, money.EUR) // Example conversion
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: *events.NewCurrencyConversionRequested(
				events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "deposit",
					UserID:        userID,
					AccountID:     sourceAccountID,
					CorrelationID: correlationID,
				},
				dr,
				events.WithConversionAmount(amount),
				events.WithConversionTo(money.EUR),
				events.WithConversionTransactionID(dr.TransactionID),
			),
			TransactionID:   dr.TransactionID,
			ConvertedAmount: convertedAmount,
		}

		// 3. Create DepositCurrencyConverted
		depositConverted := events.NewDepositCurrencyConverted(cc)

		// 4. Create DepositBusinessValidated
		dv := events.NewDepositValidated(depositConverted)

		// 5. Publish and test the flow
		err := bus.Emit(context.Background(), dv)
		require.NoError(t, err)

		// Verify the event structure
		or, ok := dv.OriginalRequest.(*events.DepositRequested)
		if !ok {
			t.Fatalf("expected DepositRequested, got %T", dv.OriginalRequest)
		}
		dcc := dv.DepositCurrencyConverted
		assert.Equal(t, userID, or.UserID)
		assert.Equal(t, sourceAccountID, or.AccountID)
		assert.True(t, dcc.ConvertedAmount.Equals(convertedAmount))
	})

	t.Run("withdraw flow", func(t *testing.T) {
		// 1. Create WithdrawRequested
		wr := events.NewWithdrawRequested(
			userID,
			sourceAccountID,
			correlationID,
			events.WithWithdrawAmount(amount),
			events.WithWithdrawID(uuid.New()),
		)

		// 3. Create WithdrawCurrencyConverted
		convertedAmount, _ := money.New(90.0, money.EUR) // Example conversion
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: *events.NewCurrencyConversionRequested(
				events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "withdraw",
					UserID:        userID,
					AccountID:     sourceAccountID,
					CorrelationID: correlationID,
				},
				wr,
				events.WithConversionAmount(amount),
				events.WithConversionTo(money.EUR),
				events.WithConversionTransactionID(wr.TransactionID),
			),
			ConvertedAmount: convertedAmount,
			TransactionID:   wr.TransactionID,
		}
		wcc := events.NewWithdrawCurrencyConverted(cc)

		// 4. Create WithdrawBusinessValidated
		wv := events.NewWithdrawValidated(wcc)

		// 5. Publish and test the flow
		err := bus.Emit(context.Background(), wv)
		require.NoError(t, err)

		// Verify the event structure
		or, ok := wv.OriginalRequest.(*events.WithdrawRequested)
		if !ok {
			t.Fatalf("expected WithdrawRequested, got %T", wv.OriginalRequest)
		}
		// wv embeds WithdrawCurrencyConverted, so we can use it directly
		t.Logf(
			"Expected converted amount: %+v, Actual converted amount: %+v",
			convertedAmount,
			wv.ConvertedAmount,
		)
		t.Logf(
			"Expected amount: %d %s, Actual amount: %d %s",
			convertedAmount.Amount(),
			convertedAmount.Currency(),
			wv.ConvertedAmount.Amount(),
			wv.ConvertedAmount.Currency(),
		)
		assert.True(t, wv.ConvertedAmount.Equals(convertedAmount))
		assert.Equal(t, sourceAccountID, or.AccountID)
	})

	t.Run("transfer flow", func(t *testing.T) {
		// 1. Create TransferRequested
		tr := &events.TransferRequested{
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

		// 2. Create CurrencyConverted
		convertedAmount, _ := money.New(90.0, money.EUR) // Example conversion
		cc := &events.CurrencyConverted{
			CurrencyConversionRequested: *events.NewCurrencyConversionRequested(
				events.FlowEvent{
					ID:            uuid.New(),
					FlowType:      "transfer",
					UserID:        userID,
					AccountID:     sourceAccountID,
					CorrelationID: correlationID,
				},
				tr,
				events.WithConversionAmount(amount),
				events.WithConversionTo(money.EUR),
				events.WithConversionTransactionID(tr.TransactionID),
			),
			TransactionID:   tr.TransactionID,
			ConvertedAmount: convertedAmount,
			ConversionInfo:  nil,
		}

		// 3. Create TransferCurrencyConverted
		tc := events.NewTransferCurrencyConverted(cc)

		// 4. Create TransferValidated
		tv := events.NewTransferBusinessValidated(tc)

		// 4. Publish and test the flow
		err := bus.Emit(context.Background(), tv)
		require.NoError(t, err)

		// Verify the event structure
		assert.Equal(t, userID, tv.UserID)
		assert.Equal(t, sourceAccountID, tv.AccountID)
		assert.Equal(t, destAccountID, tv.OriginalRequest.(*events.TransferRequested).DestAccountID)
		assert.True(t, tv.ConvertedAmount.Equals(convertedAmount))
	})
}
