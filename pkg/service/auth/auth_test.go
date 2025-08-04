package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/repository"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestCheckPasswordHash(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := authsvc.NewService(nil, nil, slog.Default())
	if !s.CheckPasswordHash("password", string(hash)) {
		t.Error("expected password to match hash")
	}
	if s.CheckPasswordHash("wrong", string(hash)) {
		t.Error("expected wrong password to not match hash")
	}
}

func TestValidEmail(t *testing.T) {
	t.Parallel()
	s := authsvc.NewService(nil, nil, slog.Default())
	if !s.ValidEmail("fixtures@example.com") {
		t.Error("expected valid email")
	}
	if s.ValidEmail("not-an-email") {
		t.Error("expected invalid email")
	}
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	assert := assert.New(t)
	uow := mocks.NewMockUnitOfWork(t)
	userRepo := mocks.NewMockUserRepository(t)
	logger := slog.Default()
	u, _ := user.NewUser("test", "bob@example.com", "password")

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	).Once()
	uow.EXPECT().UserRepository().Return(userRepo, nil).Once()
	userRepo.EXPECT().GetByUsername(u.Username).Return(u, nil).Once()

	svc := authsvc.NewBasicAuthService(uow, logger)
	loggedInUser, err := svc.Login(
		context.Background(),
		u.Username,
		"password",
	) // password matches hash
	require.NoError(err)
	assert.NotNil(loggedInUser)
	assert.Equal(u.Username, loggedInUser.Username)
}

func TestLogin_InvalidPassword(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	assert := assert.New(t)
	uow := mocks.NewMockUnitOfWork(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"wrong").Return(nil, errors.New("invalid password")).Once()
	s := authsvc.NewService(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user@example.com", "wrong")
	require.Error(err)
	assert.Nil(gotUser)

}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	uow := mocks.NewMockUnitOfWork(t)

	authStrategy := mocks.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"notfound@example.com",
		"password").Return(nil, errors.New("user not found")).Once()
	s := authsvc.NewService(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "notfound@example.com", "password")
	assert.Nil(gotUser)
	require.Error(err)
}

func TestLogin_JWTSignError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"password").Return(nil, errors.New("JWT sign error")).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user@example.com", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_GetByEmailError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"password").Return(nil, expectedErr).Once()

	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user@example.com", "password")
	require.Error(err)
	assert.Equal(expectedErr, err)
	assert.Nil(gotUser)
}

func TestGetCurrentUserId_InvalidToken(t *testing.T) {
	t.Parallel()
	uow := mocks.NewMockUnitOfWork(t)
	logger := slog.Default()
	jwtStrategy := authsvc.NewJWTAuthStrategy(uow, config.JwtConfig{}, logger)
	s := authsvc.NewService(uow, jwtStrategy, logger)
	token := &jwt.Token{}
	_, err := s.GetCurrentUserId(token)
	require.Error(t, err)
}

func TestGetCurrentUserId_MissingClaim(t *testing.T) {
	t.Parallel()
	uow := mocks.NewMockUnitOfWork(t)
	logger := slog.Default()
	jwtStrategy := authsvc.NewJWTAuthStrategy(uow, config.JwtConfig{}, logger)
	s := authsvc.NewService(uow, jwtStrategy, logger)
	token := jwt.New(jwt.SigningMethodHS256)
	_, err := s.GetCurrentUserId(token)
	require.Error(t, err)
}

func TestLogin_BasicAuthSuccess(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:       uuid.New(),
		Username: "user",
		Email:    "user@example.com",
		Password: string(hash),
	}
	authStrategy := mocks.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(user, nil).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.NoError(err)
	assert.NotNil(gotUser)
}

func TestLogin_BasicAuthInvalidPassword(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"wrong").Return(nil, nil).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "wrong")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthUoWFactoryError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	expectedErr := errors.New("uow error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(nil, expectedErr).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthUserNotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"notfound",
		"password").
		Return(nil, nil).Once()
	authStrategy.EXPECT().Login(
		mock.Anything,
		"notfound@example.com",
		"password").
		Return(nil, nil).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "notfound", "password")
	require.Error(err)
	assert.Nil(gotUser)

	gotUser, err = s.Login(context.Background(), "notfound@example.com", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_RepoErrorWithUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(nil, expectedErr).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthRepoErrorWithUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewMockAuthStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(nil, expectedErr).Once()
	s := authsvc.NewService(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.Error(err)
	assert.Nil(gotUser)
}
