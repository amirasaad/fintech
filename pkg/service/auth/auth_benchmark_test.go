package auth_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func BenchmarkCheckPasswordHash(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := authsvc.NewAuthService(nil, nil, slog.Default())
	for b.Loop() {
		s.CheckPasswordHash("password", string(hash))
	}
}

func BenchmarkLogin_Success(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := mocks.NewMockUserRepository(b)

	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Maybe()
	uow := mocks.NewMockUnitOfWork(b)
	uow.EXPECT().UserRepository().Return(repo, nil).Maybe()
	authStrategy := mocks.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login(mock.Anything, "user@example.com", "password").Return(user, nil).Maybe()
	s := authsvc.NewAuthService(uow, authStrategy, slog.Default())

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "user@example.com", "password")
	}
}

func BenchmarkValidEmail(b *testing.B) {
	s := authsvc.NewAuthService(nil, nil, slog.Default())
	for b.Loop() {
		_ = s.ValidEmail("user@example.com")
	}
}

func BenchmarkLogin_InvalidPassword(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := mocks.NewMockUserRepository(b)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Maybe()
	uow := mocks.NewMockUnitOfWork(b)
	uow.EXPECT().UserRepository().Return(repo, nil).Maybe()
	authStrategy := mocks.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login(mock.Anything, "user@example.com", "wrong").Return(nil, errors.New("invalid password")).Maybe()

	s := authsvc.NewAuthService(uow, authStrategy, slog.Default())

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "user@example.com", "wrong")
	}
}

func BenchmarkLogin_UserNotFound(b *testing.B) {
	repo := mocks.NewMockUserRepository(b)
	repo.EXPECT().GetByEmail("notfound@example.com").Return(nil, errors.New("user not found")).Maybe()
	uow := mocks.NewMockUnitOfWork(b)
	uow.EXPECT().GetRepository(mock.Anything).Return(repo, nil).Maybe()
	authStrategy := mocks.NewMockAuthStrategy(b)
	authStrategy.EXPECT().Login(mock.Anything, "notfound@example.com", "password").Return(nil, errors.New("user not found")).Maybe()

	s := authsvc.NewAuthService(uow, authStrategy, slog.Default())
	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "notfound@example.com", "password")
	}
}
