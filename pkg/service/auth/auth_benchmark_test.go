package auth_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func BenchmarkCheckPasswordHash(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := &authsvc.AuthService{}
	for b.Loop() {
		s.CheckPasswordHash("password", string(hash))
	}
}

func BenchmarkLogin_Success(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(b)

	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := fixtures.NewMockUnitOfWork(b)
	uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil).Once()
	authStrategy := fixtures.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login(mock.Anything, "user@example.com", "password").Return(user, nil).Maybe()
	s := authsvc.NewAuthService(uow, authStrategy, slog.Default())

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "user@example.com", "password")
	}
}

func BenchmarkValidEmail(b *testing.B) {
	s := &authsvc.AuthService{}
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
	uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil).Maybe()
	authStrategy := fixtures.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login(mock.Anything, "user@example.com", "wrong").Return(nil, errors.New("invalid password")).Maybe()

	s := authsvc.NewAuthService(uow, authStrategy, slog.Default())

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "user@example.com", "wrong")
	}
}

func BenchmarkLogin_UserNotFound(b *testing.B) {
	repo := fixtures.NewMockUserRepository(b)
	repo.EXPECT().GetByEmail("notfound@example.com").Return(&domain.User{}, errors.New("user not found")).Maybe()
	uow := fixtures.NewMockUnitOfWork(b)
	uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil).Maybe()
	authStrategy := fixtures.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login(mock.Anything, "notfound@example.com", "password").Return(nil, nil).Maybe()

	s := authsvc.NewAuthService(uow, authStrategy, slog.Default())
	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "notfound@example.com", "password")
	}
}
