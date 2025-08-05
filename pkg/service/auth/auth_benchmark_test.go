package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	userrepo "github.com/amirasaad/fintech/pkg/repository/user"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/dto"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func BenchmarkCheckPasswordHash(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := authsvc.New(nil, nil, slog.Default())
	for b.Loop() {
		s.CheckPasswordHash("password", string(hash))
	}
}

func BenchmarkLogin_Success(b *testing.B) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	u := &dto.UserRead{
		ID:       uuid.New(),
		Username: "user",
		Email:    "user@example.com",
	}
	repo := mocks.NewUserRepository(b)

	repo.EXPECT().GetByEmail(context.Background(), "user@example.com").Return(u, nil).Maybe()
	uow := mocks.NewUnitOfWork(b)
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(repo, nil).Maybe()
	authStrategy := mocks.NewStrategy(b)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"password").Return(&domain.User{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Password: string(hash),
	}, nil).Maybe()
	s := authsvc.New(uow, authStrategy, slog.Default())

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "user@example.com", "password")
	}
}

func BenchmarkValidEmail(b *testing.B) {
	s := authsvc.New(nil, nil, slog.Default())
	for b.Loop() {
		_ = s.ValidEmail("user@example.com")
	}
}

func BenchmarkLogin_InvalidPassword(b *testing.B) {
	u := &dto.UserRead{
		ID:       uuid.New(),
		Username: "user",
		Email:    "user@example.com",
	}
	repo := mocks.NewUserRepository(b)
	repo.EXPECT().GetByEmail(context.Background(), "user@example.com").Return(u, nil).Maybe()
	uow := mocks.NewUnitOfWork(b)
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(repo, nil).Maybe()
	authStrategy := mocks.NewStrategy(b)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"wrong",
	).Return(nil, errors.New("invalid password")).Maybe()

	s := authsvc.New(uow, authStrategy, slog.Default())

	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "user@example.com", "wrong")
	}
}

func BenchmarkLogin_UserNotFound(b *testing.B) {
	repo := mocks.NewUserRepository(b)
	repo.EXPECT().
		GetByEmail(
			context.Background(),
			"notfound@example.com",
		).Return(
		nil,
		errors.New("user not found"),
	).Maybe()
	uow := mocks.NewUnitOfWork(b)
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(repo, nil).Maybe()
	authStrategy := mocks.NewStrategy(b)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"notfound@example.com",
		"password",
	).Return(nil, errors.New("user not found")).Maybe()

	s := authsvc.New(uow, authStrategy, slog.Default())
	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Login(b.Context(), "notfound@example.com", "password")
	}
}
