package account_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func BenchmarkCreateAccount(b *testing.B) {
	require := require.New(b)
	uow := mocks.NewMockUnitOfWork(b)
	accountRepo := mocks.NewAccountRepository(b)
	uow.EXPECT().GetRepository(mock.Anything).Return(accountRepo, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	accountRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(&dto.AccountRead{}, nil)
	svc := account.New(mocks.NewMockBus(b), uow, slog.Default())
	accountRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	userID := uuid.New()
	b.ResetTimer()
	for b.Loop() {
		_, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
		require.NoError(err)

	}
}
