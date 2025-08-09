package events_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/events"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDepositRequested(t *testing.T) {
	t.Run(
		"creates deposit requested event with correct type",
		func(
			t *testing.T,
		) {
			amount, err := money.New(1000, "USD")
			require.NoError(t, err)

			e := events.DepositRequested{
				FlowEvent: events.FlowEvent{
					ID:        uuid.New(),
					FlowType:  "deposit",
					UserID:    uuid.New(),
					AccountID: uuid.New(),
				},
				Amount:        amount,
				Source:        "bank_transfer",
				TransactionID: uuid.New(),
			}

			assert.Equal(t, "Deposit.Requested", e.Type())
		})
}

func TestDepositCurrencyConverted(t *testing.T) {
	t.Run(
		"creates deposit currency converted event with correct type",
		func(
			t *testing.T,
		) {
			amount, err := money.New(1000, "USD")
			require.NoError(t, err)

			e := events.DepositCurrencyConverted{
				CurrencyConverted: events.CurrencyConverted{
					CurrencyConversionRequested: events.CurrencyConversionRequested{
						FlowEvent: events.FlowEvent{
							ID:        uuid.New(),
							FlowType:  "deposit",
							UserID:    uuid.New(),
							AccountID: uuid.New(),
						},
						Amount: amount,
						To:     "EUR",
					},
					TransactionID:   uuid.New(),
					ConvertedAmount: amount,
					ConversionInfo: &common.ConversionInfo{
						OriginalAmount:    1000.0,
						OriginalCurrency:  "USD",
						ConvertedAmount:   850.0,
						ConvertedCurrency: "EUR",
						ConversionRate:    0.85,
					},
				},
			}

			assert.Equal(t, "Deposit.CurrencyConverted", e.Type())
		})
}

func TestDepositValidated(t *testing.T) {
	t.Run("creates deposit validated event with correct type", func(t *testing.T) {
		amount, err := money.New(1000, "USD")
		require.NoError(t, err)

		e := events.DepositValidated{
			DepositCurrencyConverted: events.DepositCurrencyConverted{
				CurrencyConverted: events.CurrencyConverted{
					CurrencyConversionRequested: events.CurrencyConversionRequested{
						FlowEvent: events.FlowEvent{
							ID:        uuid.New(),
							FlowType:  "deposit",
							UserID:    uuid.New(),
							AccountID: uuid.New(),
						},
						Amount: amount,
						To:     "EUR",
					},
					TransactionID:   uuid.New(),
					ConvertedAmount: amount,
					ConversionInfo: &common.ConversionInfo{
						OriginalAmount:    1000.0,
						OriginalCurrency:  "USD",
						ConvertedAmount:   850.0,
						ConvertedCurrency: "EUR",
						ConversionRate:    0.85,
					},
				},
			},
		}

		assert.Equal(t, "Deposit.Validated", e.Type())
	})
}

func TestDepositFailed(t *testing.T) {
	t.Run(
		"creates deposit failed event with correct type and reason",
		func(t *testing.T) {
			amount, err := money.New(1000, "USD")
			require.NoError(t, err)

			e := events.DepositFailed{
				DepositRequested: events.DepositRequested{
					FlowEvent: events.FlowEvent{
						ID:        uuid.New(),
						FlowType:  "deposit",
						UserID:    uuid.New(),
						AccountID: uuid.New(),
					},
					Amount: amount, Source: "bank_transfer",
					TransactionID: uuid.New(),
				},
				Reason: "insufficient_funds",
			}

			assert.Equal(t, "Deposit.Failed", e.Type())
		})
}
