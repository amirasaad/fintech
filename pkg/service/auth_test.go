package service

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestCheckPasswordHash(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := &AuthService{logger: slog.Default()}
	if !s.CheckPasswordHash("password", string(hash)) {
		t.Error("expected password to match hash")
	}
	if s.CheckPasswordHash("wrong", string(hash)) {
		t.Error("expected wrong password to not match hash")
	}
}

func TestValidEmail(t *testing.T) {
	t.Parallel()
	s := &AuthService{logger: slog.Default()}
	if !s.ValidEmail("fixtures@example.com") {
		t.Error("expected valid email")
	}
	if s.ValidEmail("not-an-email") {
		t.Error("expected invalid email")
	}
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().GenerateToken(user).Return("testtoken", nil)
	authStrategy.EXPECT().Login("user@example.com", "password").Return(user, nil).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user@example.com", "password")
	assert.NoError(err)
	token, err := s.GenerateToken(gotUser)
	assert.NoError(err)
	assert.NotNil(gotUser)
	assert.Equal("testtoken", token)
}

func TestLogin_InvalidPassword(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	assert := assert.New(t)
	uow := fixtures.NewMockUnitOfWork(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login("user@example.com", "wrong").Return(nil, errors.New("invalid password")).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user@example.com", "wrong")
	require.Error(err)
	assert.Nil(gotUser)

}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	uow := fixtures.NewMockUnitOfWork(t)

	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login("notfound@example.com", "password").Return(nil, errors.New("user not found")).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("notfound@example.com", "password")
	assert.Nil(gotUser)
	assert.Error(err)
}

func TestLogin_JWTSignError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login("user@example.com", "password").Return(nil, errors.New("JWT sign error")).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user@example.com", "password")
	assert.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_GetByEmailError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login("user@example.com", "password").Return(nil, expectedErr).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user@example.com", "password")
	assert.Error(err)
	assert.Equal(expectedErr, err)
	assert.Nil(gotUser)
}

func TestGetCurrentUserId_InvalidToken(t *testing.T) {
	t.Parallel()
	s := &AuthService{
		strategy: &JWTAuthStrategy{logger: slog.Default()},
		logger:   slog.Default(),
	}
	token := &jwt.Token{}
	_, err := s.GetCurrentUserId(token)
	assert.Error(t, err)
}

func TestGetCurrentUserId_MissingClaim(t *testing.T) {
	t.Parallel()
	s := &AuthService{strategy: &JWTAuthStrategy{logger: slog.Default()}, logger: slog.Default()}
	token := jwt.New(jwt.SigningMethodHS256)
	_, err := s.GetCurrentUserId(token)
	assert.Error(t, err)
}

func TestLogin_BasicAuthSuccess(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login("user", "password").Return(user, nil).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user", "password")
	assert.NoError(err)
	assert.NotNil(gotUser)
}

func TestLogin_BasicAuthInvalidPassword(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login("user", "wrong").Return(nil, nil).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user", "wrong")
	assert.NoError(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthUoWFactoryError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	expectedErr := errors.New("uow error")
	authStrategy.EXPECT().Login("user", "password").Return(nil, expectedErr).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user", "password")
	assert.Error(err)
	assert.Equal(expectedErr, err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthUserNotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	authStrategy.EXPECT().Login("notfound", "password").Return(nil, nil).Once()
	authStrategy.EXPECT().Login("notfound@example.com", "password").Return(nil, nil).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("notfound", "password")
	assert.NoError(err)
	assert.Nil(gotUser)

	gotUser, err = s.Login("notfound@example.com", "password")
	assert.NoError(err)
	assert.Nil(gotUser)
}

func TestLogin_RepoErrorWithUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login("user", "password").Return(nil, expectedErr).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user", "password")
	assert.Error(err)
	assert.Equal(expectedErr, err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthRepoErrorWithUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	authStrategy := fixtures.NewMockAuthStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login("user", "password").Return(nil, expectedErr).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, nil }, authStrategy, slog.Default())
	gotUser, err := s.Login("user", "password")
	assert.Error(err)
	assert.Equal(expectedErr, err)
	assert.Nil(gotUser)
}
