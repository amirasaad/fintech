package user_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateUser(b *testing.B) {
	svc, userRepo, uow := newUserServiceWithMocks(b)
	uow.EXPECT().GetRepository(mock.Anything).Return(userRepo, nil).Maybe()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).Maybe()
	userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.CreateUser(context.Background(), "benchuser", "bench@example.com", "password")
	}
}
