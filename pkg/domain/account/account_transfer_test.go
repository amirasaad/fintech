package account

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount_Transfer(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	type accountArgs struct {
		userID   uuid.UUID
		currency currency.Code
		balance  int64
	}

	newAccount := func(userID uuid.UUID, curr currency.Code, bal int64) *Account {
		account, _ := New().WithUserID(userID).WithCurrency(curr).WithBalance(bal).Build()
		return account
	}

	cases := []struct {
		name         string
		initiator    uuid.UUID
		sourceArgs   *accountArgs
		destArgs     *accountArgs
		amount       money.Money
		expectedErr  error
		expectSrcBal money.Money
		expectDstBal money.Money
	}{
		{
			name:         "success: owner transfers to another account",
			initiator:    uuid.New(),
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:     &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:       func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(50, currency.USD); return m }(),
			expectedErr:  nil,
			expectSrcBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(50, currency.USD); return m }(),
			expectDstBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(50, currency.USD); return m }(),
		},
		// ... add other cases as needed ...
	}

	callTransfer := func(initiator uuid.UUID, source, dest *Account, amount money.Money) error {
		_, _, err := source.Transfer(initiator, dest, amount)
		return err
	}

	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var source, dest *Account
			if tc.sourceArgs != nil {
				source = newAccount(tc.sourceArgs.userID, tc.sourceArgs.currency, tc.sourceArgs.balance)
			}
			if tc.destArgs != nil {
				dest = newAccount(tc.destArgs.userID, tc.destArgs.currency, tc.destArgs.balance)
			}
			// For the first test case, use the source account owner as the initiator
			initiator := tc.initiator
			if tc.name == "success: owner transfers to another account" {
				initiator = tc.sourceArgs.userID
			}
			err := callTransfer(initiator, source, dest, tc.amount)
			require.Equal(tc.expectedErr, err)
			if source != nil {
				assert.True(tc.expectSrcBal.Equals(source.Balance))
			}
			if dest != nil {
				assert.True(tc.expectDstBal.Equals(dest.Balance))
			}
		})
	}
}
