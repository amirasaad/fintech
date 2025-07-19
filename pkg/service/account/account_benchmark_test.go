package account_test

import (
	"context"
	"testing"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func BenchmarkCreateAccount(b *testing.B) {
	require := require.New(b)
	svc, accountRepo, _, uow := newServiceWithMocks(b)
	uow.On("Do", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(repository.UnitOfWork) error)
		_ = fn(uow)
	})
	accountRepo.On("Create", mock.Anything).Return(nil)
	userID := uuid.New()
	b.ResetTimer()
	for b.Loop() {
		_, err := svc.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
		require.NoError(err)

	}
}
