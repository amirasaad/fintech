package account

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/amirasaad/fintech/pkg/domain/money"
	"github.com/amirasaad/fintech/pkg/queries"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetAccountQueryHandler_HandleQuery(t *testing.T) {
	validUser := uuid.New()
	validAccount := uuid.New()
	m, err := money.New(100.0, "USD")
	if err != nil {
		panic(err)
	}
	acc := &account.Account{ID: validAccount, UserID: validUser, Balance: m}

	tests := []struct {
		name         string
		query        queries.GetAccountQuery
		setupMock    func(*mocks.MockUnitOfWork, *mocks.MockAccountRepository)
		expectErr    bool
		wantNil      bool
		expectEvents int
	}{
		{
			name:  "valid account and user",
			query: queries.GetAccountQuery{AccountID: validAccount.String(), UserID: validUser.String()},
			setupMock: func(uow *mocks.MockUnitOfWork, repo *mocks.MockAccountRepository) {
				uow.On("AccountRepository").Return(repo, nil)
				repo.On("Get", validAccount).Return(acc, nil)
			},
			expectErr:    false,
			wantNil:      false,
			expectEvents: 1, // AccountQuerySucceededEvent
		},
		{
			name:  "not found",
			query: queries.GetAccountQuery{AccountID: uuid.New().String(), UserID: validUser.String()},
			setupMock: func(uow *mocks.MockUnitOfWork, repo *mocks.MockAccountRepository) {
				uow.On("AccountRepository").Return(repo, nil)
				repo.On("Get", mock.AnythingOfType("uuid.UUID")).Return(nil, errors.New("not found"))
			},
			expectErr:    true,
			wantNil:      true,
			expectEvents: 1, // AccountQueryFailedEvent
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uow := mocks.NewMockUnitOfWork(t)
			repo := mocks.NewMockAccountRepository(t)
			bus := &mockEventBus{}
			tc.setupMock(uow, repo)

			handler := GetAccountQueryHandler(uow, bus)
			result, err := handler.HandleQuery(context.Background(), tc.query)

			// Check events were published
			// assert.Len(t, bus.published, tc.expectEvents)

			if tc.expectErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				res, ok := result.(queries.GetAccountResult)
				assert.True(t, ok)
				if tc.wantNil {
					assert.Empty(t, res.AccountID)
					assert.Empty(t, res.UserID)
					assert.Zero(t, res.Balance)
					assert.Empty(t, res.Currency)
				} else {
					assert.Equal(t, acc.ID.String(), res.AccountID)
					assert.Equal(t, acc.UserID.String(), res.UserID)
					// Optionally check Balance and Currency if set in the handler
				}
			}
		})
	}
}
