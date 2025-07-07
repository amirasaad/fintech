package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func BenchmarkCreateUser(b *testing.B) {
	svc, userRepo, uow := newUserServiceWithMocks(b)
	uow.EXPECT().Begin().Return(nil).Maybe()
	uow.EXPECT().Commit().Return(nil).Maybe()
	userRepo.EXPECT().Create(mock.Anything).Return(nil).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.CreateUser("benchuser", "bench@example.com", "password")
	}
}

func BenchmarkValidUser(b *testing.B) {
	svc, userRepo, _ := newUserServiceWithMocks(b)
	id := uuid.New()
	userRepo.EXPECT().Valid(id, "password").Return(true).Maybe()
	b.ResetTimer()
	for b.Loop() {
		_, _ = svc.ValidUser(id, "password")
	}
}
