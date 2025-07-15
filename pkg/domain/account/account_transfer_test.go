package account

import (
	"testing"
	"time"

	"github.com/amirasaad/fintech/pkg/currency"
	"github.com/amirasaad/fintech/pkg/domain/common"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccount_Transfer(t *testing.T) {
	require := require.New(t)
	usd := currency.USD

	type eventExpectation struct {
		wantEvent bool
		event     string
	}

	type testCase struct {
		name             string
		sourceAccount    *Account
		destAccount      *Account
		sourceUserID     uuid.UUID
		destUserID       uuid.UUID
		amount           money.Money
		expectedErr      error
		eventExpectation eventExpectation
	}

	sameUUID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	userA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	accA := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	accB := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	testCases := []testCase{
		{
			name:          "success: owner transfers to another account",
			sourceAccount: NewAccountFromData(accA, userA, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:   NewAccountFromData(accB, userB, money.NewFromData(0, string(usd)), time.Time{}, time.Time{}),
			sourceUserID:  userA,
			destUserID:    userB,
			amount:        money.NewFromData(50, string(usd)),
			expectedErr:   nil,
			eventExpectation: eventExpectation{
				wantEvent: true,
				event:     "TransferRequestedEvent",
			},
		},
		{
			name:             "fail: cannot transfer to same account",
			sourceAccount:    NewAccountFromData(sameUUID, sameUUID, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:      NewAccountFromData(sameUUID, sameUUID, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			sourceUserID:     sameUUID,
			destUserID:       sameUUID,
			amount:           money.NewFromData(10, string(usd)),
			expectedErr:      ErrCannotTransferToSameAccount,
			eventExpectation: eventExpectation{wantEvent: false},
		},
		{
			name:             "fail: nil dest account",
			sourceAccount:    NewAccountFromData(accA, userA, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:      nil,
			sourceUserID:     userA,
			destUserID:       uuid.Nil,
			amount:           money.NewFromData(10, string(usd)),
			expectedErr:      ErrNilAccount,
			eventExpectation: eventExpectation{wantEvent: false},
		},
		{
			name:             "fail: not owner",
			sourceAccount:    NewAccountFromData(accA, userA, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:      NewAccountFromData(accB, userB, money.NewFromData(0, string(usd)), time.Time{}, time.Time{}),
			sourceUserID:     userB,
			destUserID:       userA,
			amount:           money.NewFromData(10, string(usd)),
			expectedErr:      ErrNotOwner,
			eventExpectation: eventExpectation{wantEvent: false},
		},
		{
			name:             "fail: negative amount",
			sourceAccount:    NewAccountFromData(accA, userA, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:      NewAccountFromData(accB, userB, money.NewFromData(0, string(usd)), time.Time{}, time.Time{}),
			sourceUserID:     userA,
			destUserID:       userB,
			amount:           money.NewFromData(-10, string(usd)),
			expectedErr:      ErrTransactionAmountMustBePositive,
			eventExpectation: eventExpectation{wantEvent: false},
		},
		{
			name:             "fail: zero amount",
			sourceAccount:    NewAccountFromData(accA, userA, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:      NewAccountFromData(accB, userB, money.NewFromData(0, string(usd)), time.Time{}, time.Time{}),
			sourceUserID:     userA,
			destUserID:       userB,
			amount:           money.NewFromData(0, string(usd)),
			expectedErr:      ErrTransactionAmountMustBePositive,
			eventExpectation: eventExpectation{wantEvent: false},
		},
		{
			name:             "fail: currency mismatch",
			sourceAccount:    NewAccountFromData(accA, userA, money.NewFromData(100, string(usd)), time.Time{}, time.Time{}),
			destAccount:      NewAccountFromData(accB, userB, money.NewFromData(0, string(currency.EUR)), time.Time{}, time.Time{}),
			sourceUserID:     userA,
			destUserID:       userB,
			amount:           money.NewFromData(10, string(currency.EUR)),
			expectedErr:      ErrCurrencyMismatch,
			eventExpectation: eventExpectation{wantEvent: false},
		},
		{
			name:             "fail: insufficient funds",
			sourceAccount:    NewAccountFromData(accA, userA, money.NewFromData(5, string(usd)), time.Time{}, time.Time{}),
			destAccount:      NewAccountFromData(accB, userB, money.NewFromData(0, string(usd)), time.Time{}, time.Time{}),
			sourceUserID:     userA,
			destUserID:       userB,
			amount:           money.NewFromData(10, string(usd)),
			expectedErr:      ErrInsufficientFunds,
			eventExpectation: eventExpectation{wantEvent: false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.sourceAccount.Transfer(tc.sourceUserID, tc.destUserID, tc.destAccount, tc.amount, MoneySourceCard)
			require.ErrorIs(err, tc.expectedErr)
			events := tc.sourceAccount.PullEvents()

			if tc.eventExpectation.wantEvent {
				require.Len(events, 1)
				evt := events[0]
				require.EqualValues(tc.eventExpectation.event, evt.EventType())
			}
			destEvents := []common.Event{}
			if tc.destAccount != nil && tc.destAccount != tc.sourceAccount {
				destEvents = tc.destAccount.PullEvents()
			}
			require.Len(destEvents, 0)
		})
	}
}
