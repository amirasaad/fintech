package account

import (
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mustMoney(m money.Money, err error) money.Money {
	if err != nil {
		panic(err)
	}
	return m
}

func TestAccount_Transfer(t *testing.T) {
	require := require.New(t)

	type accountArgs struct {
		userID   uuid.UUID
		currency currency.Code
		balance  int64
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
		useSameAcct  bool
		sourceNil    bool
		destNil      bool
	}{
		{
			name:         "success: owner transfers to another account",
			initiator:    uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:     &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:       mustMoney(money.NewMoneyFromSmallestUnit(50, currency.USD)),
			expectedErr:  nil,
			expectSrcBal: mustMoney(money.NewMoneyFromSmallestUnit(50, currency.USD)),
			expectDstBal: mustMoney(money.NewMoneyFromSmallestUnit(50, currency.USD)),
		},
		{
			name:         "fail: cannot transfer to same account",
			initiator:    uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:     &accountArgs{userID: uuid.Nil, currency: currency.USD, balance: 0},
			amount:       mustMoney(money.NewMoneyFromSmallestUnit(10, currency.USD)),
			expectedErr:  ErrCannotTransferToSameAccount,
			expectSrcBal: mustMoney(money.NewMoneyFromSmallestUnit(100, currency.USD)),
			expectDstBal: mustMoney(money.NewMoneyFromSmallestUnit(100, currency.USD)),
			useSameAcct:  true,
		},
		{
			name:        "fail: nil source account",
			initiator:   uuid.New(),
			sourceArgs:  nil,
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      mustMoney(money.NewMoneyFromSmallestUnit(10, currency.USD)),
			expectedErr: ErrNilAccount,
			sourceNil:   true,
		},
		{
			name:        "fail: nil dest account",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    nil,
			amount:      mustMoney(money.NewMoneyFromSmallestUnit(10, currency.USD)),
			expectedErr: ErrNilAccount,
			destNil:     true,
		},
		{
			name:        "fail: not owner",
			initiator:   uuid.New(), // not the source account owner
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      mustMoney(money.NewMoneyFromSmallestUnit(10, currency.USD)),
			expectedErr: ErrNotOwner,
		},
		{
			name:        "fail: negative amount",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      mustMoney(money.NewMoneyFromSmallestUnit(-10, currency.USD)),
			expectedErr: ErrTransactionAmountMustBePositive,
		},
		{
			name:        "fail: zero amount",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      mustMoney(money.NewMoneyFromSmallestUnit(0, currency.USD)),
			expectedErr: ErrTransactionAmountMustBePositive,
		},
		{
			name:        "fail: currency mismatch",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.EUR, balance: 0},
			amount:      mustMoney(money.NewMoneyFromSmallestUnit(10, currency.USD)),
			expectedErr: ErrCurrencyMismatch,
		},
		{
			name:         "fail: insufficient funds",
			initiator:    uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 5},
			destArgs:     &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:       mustMoney(money.NewMoneyFromSmallestUnit(10, currency.USD)),
			expectedErr:  ErrInsufficientFunds,
			expectSrcBal: mustMoney(money.NewMoneyFromSmallestUnit(5, currency.USD)),
			expectDstBal: mustMoney(money.NewMoneyFromSmallestUnit(0, currency.USD)),
		},
	}

	callTransfer := func(initiator uuid.UUID, source, dest *Account, amount money.Money) error {
		err := source.Transfer(initiator, dest, amount, MoneySourceCard)
		return err
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var source, dest *Account
			switch {
			case tc.sourceNil:
				source = nil
			case tc.sourceArgs != nil:
				source, _ = New().WithUserID(tc.sourceArgs.userID).WithCurrency(tc.sourceArgs.currency).WithBalance(tc.sourceArgs.balance).Build()
			}
			switch {
			case tc.destNil:
				dest = nil
			case tc.useSameAcct:
				dest = source
			case tc.destArgs != nil:
				dest, _ = New().WithUserID(tc.destArgs.userID).WithCurrency(tc.destArgs.currency).WithBalance(tc.destArgs.balance).Build()
			}
			initiator := tc.initiator
			if initiator == uuid.Nil && source != nil {
				initiator = source.UserID
			}
			err := callTransfer(initiator, source, dest, tc.amount)
			require.ErrorIs(err, tc.expectedErr)

		})
	}
}
