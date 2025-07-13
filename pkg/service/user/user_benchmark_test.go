package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateUser(b *testing.B) {
	svc, userRepo, uow := newUserServiceWithMocks(b)
	uow.EXPECT().GetRepository(mock.Anything).Return(userRepo, nil).Maybe()
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).Maybe()
	userRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.CreateUser(context.Background(), "benchuser", "bench@example.com", "password")
	}
}

func BenchmarkValidUser(b *testing.B) {
	svc, userRepo, uow := newUserServiceWithMocks(b)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).Maybe()
	id := uuid.New()
	userRepo.EXPECT().Valid(id, "password").Return(true).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.ValidUser(context.Background(), id.String(), "password")
	}
}
