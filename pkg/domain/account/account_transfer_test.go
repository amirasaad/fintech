package account

import (
	"fmt"
	"testing"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccount_Transfer(t *testing.T) {
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
		useSameAcct  bool // new: if true, use same account for source and dest
		sourceNil    bool // new: if true, source is nil
		destNil      bool // new: if true, dest is nil
	}{
		{
			name:         "success: owner transfers to another account",
			initiator:    uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:     &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:       func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(50, currency.USD); return m }(),
			expectedErr:  nil,
			expectSrcBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(50, currency.USD); return m }(),
			expectDstBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(50, currency.USD); return m }(),
		},
		{
			name:         "fail: cannot transfer to same account",
			initiator:    uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:     &accountArgs{userID: uuid.Nil, currency: currency.USD, balance: 0},
			amount:       func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(10, currency.USD); return m }(),
			expectedErr:  fmt.Errorf("cannot transfer to same account"),
			expectSrcBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(100, currency.USD); return m }(),
			expectDstBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(100, currency.USD); return m }(),
			useSameAcct:  true,
		},
		{
			name:        "fail: nil source account",
			initiator:   uuid.New(),
			sourceArgs:  nil,
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(10, currency.USD); return m }(),
			expectedErr: fmt.Errorf("nil account"),
			sourceNil:   true,
		},
		{
			name:        "fail: nil dest account",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    nil,
			amount:      func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(10, currency.USD); return m }(),
			expectedErr: fmt.Errorf("nil account"),
			destNil:     true,
		},
		{
			name:        "fail: not owner",
			initiator:   uuid.New(), // not the source account owner
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(10, currency.USD); return m }(),
			expectedErr: fmt.Errorf("not owner"),
		},
		{
			name:        "fail: negative amount",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(-10, currency.USD); return m }(),
			expectedErr: fmt.Errorf("amount must be positive"),
		},
		{
			name:        "fail: zero amount",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:      func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(0, currency.USD); return m }(),
			expectedErr: fmt.Errorf("amount must be positive"),
		},
		{
			name:        "fail: currency mismatch",
			initiator:   uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:  &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 100},
			destArgs:    &accountArgs{userID: uuid.New(), currency: currency.EUR, balance: 0},
			amount:      func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(10, currency.USD); return m }(),
			expectedErr: fmt.Errorf("currency mismatch"),
		},
		{
			name:         "fail: insufficient funds",
			initiator:    uuid.Nil, // will be set to sourceArgs.userID below
			sourceArgs:   &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 5},
			destArgs:     &accountArgs{userID: uuid.New(), currency: currency.USD, balance: 0},
			amount:       func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(10, currency.USD); return m }(),
			expectedErr:  fmt.Errorf("insufficient funds"),
			expectSrcBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(5, currency.USD); return m }(),
			expectDstBal: func() money.Money { m, _ := money.NewMoneyFromSmallestUnit(0, currency.USD); return m }(),
		},
	}

	callTransfer := func(initiator uuid.UUID, source, dest *Account, amount money.Money) error {
		_, _, err := source.Transfer(initiator, dest, amount)
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
				source = newAccount(tc.sourceArgs.userID, tc.sourceArgs.currency, tc.sourceArgs.balance)
			}
			switch {
			case tc.destNil:
				dest = nil
			case tc.useSameAcct:
				dest = source
			case tc.destArgs != nil:
				dest = newAccount(tc.destArgs.userID, tc.destArgs.currency, tc.destArgs.balance)
			}
			initiator := tc.initiator
			if initiator == uuid.Nil && source != nil {
				initiator = source.UserID
			}
			err := callTransfer(initiator, source, dest, tc.amount)
			if tc.expectedErr != nil {
				assert.EqualError(err, tc.expectedErr.Error())
			} else {
				assert.NoError(err)
			}
			if source != nil && tc.expectSrcBal.Amount() != 0 {
				assert.True(tc.expectSrcBal.Equals(source.Balance))
			}
			if dest != nil && tc.expectDstBal.Amount() != 0 {
				assert.True(tc.expectDstBal.Equals(dest.Balance))
			}
		})
	}
}
