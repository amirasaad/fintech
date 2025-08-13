package account_test

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccount_Transfer(t *testing.T) {
	require := require.New(t)
	usd := currency.USD

	type testCase struct {
		name          string
		sourceAccount *account.Account
		destAccount   *account.Account
		sourceUserID  uuid.UUID
		destUserID    uuid.UUID
		amount        money.Money
		expectedErr   error
	}

	sameUUID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	sourceAccount, err := account.New().WithUserID(userA).WithBalance(100).WithCurrency(usd).Build()
	require.NoError(err)
	destAccount, err := account.New().WithUserID(userB).WithCurrency(usd).Build()
	require.NoError(err)

	testCases := []testCase{
		{
			name:          "success: owner transfers to another account",
			sourceAccount: sourceAccount,
			destAccount:   destAccount,
			sourceUserID:  userA,
			destUserID:    userB,
			amount:        money.NewFromData(50, string(usd)),
			expectedErr:   nil,
		},
		{
			name: "fail: cannot transfer to same account",
			sourceAccount: func() *account.Account {
				acc, err := account.New().
					WithUserID(sameUUID).
					WithID(sameUUID).
					WithBalance(100).
					WithCurrency(usd).
					Build()
				require.NoError(err)
				return acc
			}(),
			destAccount: func() *account.Account {
				acc, err := account.New().
					WithUserID(sameUUID).
					WithID(sameUUID).
					WithBalance(100).
					WithCurrency(usd).
					Build()
				require.NoError(err)
				return acc
			}(),
			sourceUserID: sameUUID,
			destUserID:   sameUUID,
			amount:       money.NewFromData(10, string(usd)),
			expectedErr:  account.ErrCannotTransferToSameAccount,
		},
		{
			name:          "fail: nil dest account",
			sourceAccount: sourceAccount,
			destAccount:   nil,
			sourceUserID:  userA,
			destUserID:    uuid.Nil,
			amount:        money.NewFromData(10, string(usd)),
			expectedErr:   account.ErrNilAccount,
		},
		{
			name:          "fail: not owner",
			sourceAccount: sourceAccount,
			destAccount:   destAccount,
			sourceUserID:  userB,
			destUserID:    userA,
			amount:        money.NewFromData(10, string(usd)),
			expectedErr:   account.ErrNotOwner,
		},
		{
			name:          "fail: negative amount",
			sourceAccount: sourceAccount,
			destAccount:   destAccount,
			sourceUserID:  userA,
			destUserID:    userB,
			amount:        money.NewFromData(-10, string(usd)),
			expectedErr:   account.ErrTransactionAmountMustBePositive,
		},
		{
			name:          "fail: zero amount",
			sourceAccount: sourceAccount,
			destAccount:   destAccount,
			sourceUserID:  userA,
			destUserID:    userB,
			amount:        money.NewFromData(0, string(usd)),
			expectedErr:   account.ErrTransactionAmountMustBePositive,
		},
		{
			name: "fail: currency mismatch",
			sourceAccount: func() *account.Account {
				acc, err := account.New().
					WithUserID(userA).
					WithBalance(100).
					WithCurrency(usd).
					Build()
				require.NoError(err)
				return acc
			}(),
			destAccount: func() *account.Account {
				acc, err := account.New().
					WithUserID(userB).
					WithCurrency(currency.EUR).
					Build()
				require.NoError(err)
				return acc
			}(),
			sourceUserID: userA,
			destUserID:   userB,
			amount:       money.NewFromData(10, string(currency.EUR)),
			expectedErr:  account.ErrCurrencyMismatch,
		},
		{
			name: "fail: insufficient funds",
			sourceAccount: func() *account.Account {
				acc, err := account.New().
					WithUserID(userA).
					WithBalance(5).
					WithCurrency(usd).
					Build()
				require.NoError(err)
				return acc
			}(),
			destAccount: func() *account.Account {
				acc, err := account.New().
					WithUserID(userB).
					WithCurrency(usd).
					Build()
				require.NoError(err)
				return acc
			}(),
			sourceUserID: userA,
			destUserID:   userB,
			amount:       money.NewFromData(10, string(usd)),
			expectedErr:  account.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.sourceAccount.
				ValidateTransfer(
					tc.sourceUserID,
					tc.destUserID,
					tc.destAccount,
					tc.amount,
				)
			require.ErrorIs(err, tc.expectedErr)

		})
	}
}
