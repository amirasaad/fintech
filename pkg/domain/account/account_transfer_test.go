package account

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAccount(userID uuid.UUID, balance int64, curr string) *Account {
	return &Account{
		ID:       uuid.New(),
		UserID:   userID,
		Balance:  balance,
		Currency: currency.Code(curr),
	}
}

func callTransfer(initiatorUserID uuid.UUID, src, dst *Account, amount int64) error {
	if src != nil {
		_, _, err := src.Transfer(initiatorUserID, dst, amount)
		return err
	}
	var nilSrc *Account
	_, _, err := nilSrc.Transfer(initiatorUserID, dst, amount)
	return err
}

func TestAccount_Transfer(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	sourceUserID := uuid.New()
	destUserID := uuid.New()
	otherUserID := uuid.New()

	errNilAccount := errors.New("nil account")
	errNotOwner := errors.New("not owner")
	errNegative := errors.New("amount must be positive")
	errCurrency := errors.New("currency mismatch")
	errFunds := errors.New("insufficient funds")

	tests := []struct {
		name            string
		initiatorUserID uuid.UUID
		source          *Account
		dest            *Account
		amount          int64
		expectedErr     error
		expectSrcBal    int64
		expectDstBal    int64
	}{
		{
			name:            "success",
			initiatorUserID: sourceUserID,
			source:          newTestAccount(sourceUserID, 10000, "USD"),
			dest:            newTestAccount(destUserID, 5000, "USD"),
			amount:          2500,
			expectedErr:     nil,
			expectSrcBal:    7500,
			expectDstBal:    7500,
		},
		{
			name:            "insufficient funds",
			initiatorUserID: sourceUserID,
			source:          newTestAccount(sourceUserID, 1000, "USD"),
			dest:            newTestAccount(destUserID, 5000, "USD"),
			amount:          2500,
			expectedErr:     errFunds,
			expectSrcBal:    1000,
			expectDstBal:    5000,
		},
		{
			name:            "negative amount",
			initiatorUserID: sourceUserID,
			source:          newTestAccount(sourceUserID, 10000, "USD"),
			dest:            newTestAccount(destUserID, 5000, "USD"),
			amount:          -100,
			expectedErr:     errNegative,
			expectSrcBal:    10000,
			expectDstBal:    5000,
		},
		{
			name:            "currency mismatch",
			initiatorUserID: sourceUserID,
			source:          newTestAccount(sourceUserID, 10000, "USD"),
			dest:            newTestAccount(destUserID, 5000, "EUR"),
			amount:          1000,
			expectedErr:     errCurrency,
			expectSrcBal:    10000,
			expectDstBal:    5000,
		},
		{
			name:            "not source owner",
			initiatorUserID: otherUserID,
			source:          newTestAccount(sourceUserID, 10000, "USD"),
			dest:            newTestAccount(destUserID, 5000, "USD"),
			amount:          1000,
			expectedErr:     errNotOwner,
			expectSrcBal:    10000,
			expectDstBal:    5000,
		},
		{
			name:            "nil source",
			initiatorUserID: sourceUserID,
			source:          nil,
			dest:            newTestAccount(destUserID, 5000, "USD"),
			amount:          1000,
			expectedErr:     errNilAccount,
			expectSrcBal:    0,
			expectDstBal:    5000,
		},
		{
			name:            "nil dest",
			initiatorUserID: sourceUserID,
			source:          newTestAccount(sourceUserID, 10000, "USD"),
			dest:            nil,
			amount:          1000,
			expectedErr:     errNilAccount,
			expectSrcBal:    10000,
			expectDstBal:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := callTransfer(tc.initiatorUserID, tc.source, tc.dest, tc.amount)
			require.Equal(tc.expectedErr, err)
			if tc.source != nil {
				assert.Equal(tc.expectSrcBal, tc.source.Balance)
			}
			if tc.dest != nil {
				assert.Equal(tc.expectDstBal, tc.dest.Balance)
			}
		})
	}
}
