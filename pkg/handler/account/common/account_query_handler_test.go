package common

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
		setupMocks   func(bus *mocks.MockEventBus)
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
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.AccountQuerySucceededEvent")).Return(nil)
			},
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
			setupMocks: func(bus *mocks.MockEventBus) {
				bus.On("Publish", mock.Anything, mock.AnythingOfType("events.AccountQueryFailedEvent")).Return(nil)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uow := mocks.NewMockUnitOfWork(t)
			repo := mocks.NewMockAccountRepository(t)
			bus := mocks.NewMockEventBus(t)
			if tc.setupMocks != nil {
				tc.setupMocks(bus)
			}
			tc.setupMock(uow, repo)

			handler := GetAccountQueryHandler(uow, bus)
			result, err := handler.HandleQuery(context.Background(), tc.query)

			// Optionally, add mock assertions for event publishing here if needed

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
