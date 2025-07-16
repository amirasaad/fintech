package account

import (
	"context"
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetAccountQueryHandler_HandleQuery(t *testing.T) {
	validUser := uuid.New()
	validAccount := uuid.New()
	acc := &account.Account{ID: validAccount, UserID: validUser}

	tests := []struct {
		name         string
		query        account.GetAccountQuery
		setupMock    func(*mocks.MockUnitOfWork, *mocks.MockAccountRepository)
		expectErr    bool
		wantNil      bool
		expectEvents int
	}{
		{
			name:  "valid account and user",
			query: account.GetAccountQuery{AccountID: validAccount.String(), UserID: validUser.String()},
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
			query: account.GetAccountQuery{AccountID: uuid.New().String(), UserID: validUser.String()},
			setupMock: func(uow *mocks.MockUnitOfWork, repo *mocks.MockAccountRepository) {
				uow.On("AccountRepository").Return(repo, nil)
				repo.On("Get", mock.AnythingOfType("uuid.UUID")).Return(nil, errors.New("not found"))
			},
			expectErr:    true,
			wantNil:      true,
			expectEvents: 1, // AccountQueryFailedEvent
		},
		{
			name:  "unauthorized user",
			query: account.GetAccountQuery{AccountID: validAccount.String(), UserID: uuid.New().String()},
			setupMock: func(uow *mocks.MockUnitOfWork, repo *mocks.MockAccountRepository) {
				uow.On("AccountRepository").Return(repo, nil)
				repo.On("Get", validAccount).Return(acc, nil)
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
			assert.Len(t, bus.published, tc.expectEvents)

			if tc.expectErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				res, ok := result.(GetAccountResult)
				assert.True(t, ok)
				if tc.wantNil {
					assert.Nil(t, res.Account)
				} else {
					assert.Equal(t, acc, res.Account)
				}
			}
		})
	}
}
