package service

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func BenchmarkCheckPasswordHash(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := &AuthService{}
	for b.Loop() {
		s.CheckPasswordHash("password", string(hash))
	}
}

func BenchmarkLogin_Success(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(b)

	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Maybe()
	uow := fixtures.NewMockUnitOfWork(b)
	uow.EXPECT().UserRepository().Return(repo).Maybe()
	authStrategy := fixtures.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login("user@example.com", "password").Return(user, nil).Maybe()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil }, authStrategy)

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login("user@example.com", "password")
	}
}

func BenchmarkValidEmail(b *testing.B) {
	s := &AuthService{}
	for b.Loop() {
		_ = s.ValidEmail("user@example.com")
	}
}

func BenchmarkLogin_InvalidPassword(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(b)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Maybe()
	uow := fixtures.NewMockUnitOfWork(b)
	uow.EXPECT().UserRepository().Return(repo).Maybe()
	authStrategy := fixtures.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login("user@example.com", "wrong").Return(nil, errors.New("invalid password")).Maybe()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil }, authStrategy)

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login("user@example.com", "wrong")
	}
}

func BenchmarkLogin_UserNotFound(b *testing.B) {
	repo := fixtures.NewMockUserRepository(b)
	repo.EXPECT().GetByEmail("notfound@example.com").Return(&domain.User{}, errors.New("user not found")).Maybe()
	uow := fixtures.NewMockUnitOfWork(b)
	uow.EXPECT().UserRepository().Return(repo).Maybe()
	authStrategy := fixtures.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login("notfound@example.com", "password").Return(nil, nil).Maybe()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil }, authStrategy)
	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login("notfound@example.com", "password")
	}
}
