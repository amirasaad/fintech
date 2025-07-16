package account

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAccountService struct {
	getBalanceFn func(ctx context.Context, userID, accountID string) (float64, string, error)
}

func (m *mockAccountService) GetBalance(ctx context.Context, userID, accountID string) (float64, string, error) {
	return m.getBalanceFn(ctx, userID, accountID)
}

func TestGetAccountBalanceQueryHandler_HandleQuery(t *testing.T) {
	validUser := "user-1"
	validAccount := "acc-1"
	balance := 100.0
	currency := "USD"

	tests := []struct {
		name      string
		query     GetAccountBalanceQuery
		service   *mockAccountService
		expectErr bool
		expectBal float64
		expectCur string
	}{
		{
			name:  "valid account and user",
			query: GetAccountBalanceQuery{AccountID: validAccount, UserID: validUser},
			service: &mockAccountService{
				getBalanceFn: func(ctx context.Context, userID, accountID string) (float64, string, error) {
					assert.Equal(t, validUser, userID)
					assert.Equal(t, validAccount, accountID)
					return balance, currency, nil
				},
			},
			expectErr: false,
			expectBal: balance,
			expectCur: currency,
		},
		{
			name:  "non-existent account",
			query: GetAccountBalanceQuery{AccountID: "notfound", UserID: validUser},
			service: &mockAccountService{
				getBalanceFn: func(ctx context.Context, userID, accountID string) (float64, string, error) {
					return 0, "", errors.New("account not found")
				},
			},
			expectErr: true,
		},
		{
			name:  "unauthorized user",
			query: GetAccountBalanceQuery{AccountID: validAccount, UserID: "baduser"},
			service: &mockAccountService{
				getBalanceFn: func(ctx context.Context, userID, accountID string) (float64, string, error) {
					return 0, "", errors.New("unauthorized")
				},
			},
			expectErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := GetAccountBalanceQueryHandler(tc.service)
			result, err := handler.HandleQuery(context.Background(), tc.query)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				res, ok := result.(GetAccountBalanceResult)
				assert.True(t, ok)
				assert.InEpsilon(t, tc.expectBal, res.Balance, 0.01)
				assert.Equal(t, tc.expectCur, res.Currency)
			}
		})
	}
}
