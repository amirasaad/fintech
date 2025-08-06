package auth_test

import (
	"context"
	"errors"
	"github.com/amirasaad/fintech/pkg/config"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/dto"
	userrepo "github.com/amirasaad/fintech/pkg/repository/user"
	authsvc "github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/amirasaad/fintech/pkg/utils"
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
	s := authsvc.New(nil, nil, slog.Default())
	if !s.CheckPasswordHash("password", string(hash)) {
		t.Error("expected password to match hash")
	}
	if s.CheckPasswordHash("wrong", string(hash)) {
		t.Error("expected wrong password to not match hash")
	}
}

func TestValidEmail(t *testing.T) {
	t.Parallel()
	s := authsvc.New(nil, nil, slog.Default())
	if !s.ValidEmail("fixtures@example.com") {
		t.Error("expected valid email")
	}
	if s.ValidEmail("not-an-email") {
		t.Error("expected invalid email")
	}
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	hash, _ := utils.HashPassword("password")
	require := require.New(t)
	assert := assert.New(t)
	uow := mocks.NewUnitOfWork(t)
	userRepo := mocks.NewUserRepository(t)
	logger := slog.Default()

	// Create a test user with known credentials
	username := "testuser"
	email := "test@example.com"
	userID := uuid.New()

	// The BasicAuthStrategy uses a hardcoded dummy hash that matches the password "password"
	svc := authsvc.NewWithBasic(uow, logger)

	// Mock the repository access
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(userRepo, nil).Once()
	userRepo.EXPECT().GetByUsername(context.Background(), username).Return(&dto.UserRead{
		ID:             userID,
		Username:       username,
		Email:          email,
		HashedPassword: string(hash),
	}, nil).Once()

	// The hardcoded hash in BasicAuthStrategy is for the password "password"
	// So we need to use "password" as the password in this test
	// Note: The hash in BasicAuthStrategy is for the password "password"
	loggedInUser, err := svc.Login(
		context.Background(),
		username,
		"password", // This must match the password that was hashed to create the hardcoded hash
	)
	require.NoError(err, "Login should succeed with correct password")
	require.NotNil(loggedInUser, "Logged in user should not be nil")
	assert.Equal(username, loggedInUser.Username, "Returned user should have the correct username")
	assert.Equal(email, loggedInUser.Email, "Returned user should have the correct email")
	assert.Equal(userID, loggedInUser.ID, "Returned user should have the correct ID")
}

func TestLogin_InvalidPassword(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	assert := assert.New(t)
	uow := mocks.NewUnitOfWork(t)
	authStrategy := mocks.NewStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"wrong").Return(nil, errors.New("invalid password")).Once()
	s := authsvc.New(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user@example.com", "wrong")
	require.Error(err)
	assert.Nil(gotUser)

}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	uow := mocks.NewUnitOfWork(t)

	authStrategy := mocks.NewStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"notfound@example.com",
		"password").Return(nil, errors.New("user not found")).Once()
	s := authsvc.New(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "notfound@example.com", "password")
	assert.Nil(gotUser)
	require.Error(err)
}

func TestLogin_JWTSignError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	uow := mocks.NewUnitOfWork(t)
	authStrategy := mocks.NewStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"password").Return(nil, errors.New("JWT sign error")).Once()
	s := authsvc.New(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user@example.com", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_GetByEmailError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	uow := mocks.NewUnitOfWork(t)
	authStrategy := mocks.NewStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user@example.com",
		"password").Return(nil, expectedErr).Once()

	s := authsvc.New(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user@example.com", "password")
	require.Error(err)
	assert.Equal(expectedErr, err)
	assert.Nil(gotUser)
}

func TestGetCurrentUserId_InvalidToken(t *testing.T) {
	t.Parallel()
	uow := mocks.NewUnitOfWork(t)
	logger := slog.Default()
	jwtStrategy := authsvc.NewWithJWT(uow, config.JwtConfig{}, logger)
	s := authsvc.New(uow, jwtStrategy, logger)
	token := &jwt.Token{}
	_, err := s.GetCurrentUserId(token)
	require.Error(t, err)
}

func TestGetCurrentUserId_MissingClaim(t *testing.T) {
	t.Parallel()
	uow := mocks.NewUnitOfWork(t)
	logger := slog.Default()
	jwtStrategy := authsvc.NewWithJWT(uow, config.JwtConfig{}, logger)
	s := authsvc.New(uow, jwtStrategy, logger)
	token := jwt.New(jwt.SigningMethodHS256)
	_, err := s.GetCurrentUserId(token)
	require.Error(t, err)
}

func TestLogin_BasicAuthSuccess(t *testing.T) {
	t.Parallel()
	uow := mocks.NewUnitOfWork(t)
	assert := assert.New(t)
	require := require.New(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:       uuid.New(),
		Username: "user",
		Email:    "user@example.com",
		Password: string(hash),
	}
	authStrategy := mocks.NewStrategy(t)
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(user, nil).Once()
	s := authsvc.New(uow, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.NoError(err)
	assert.NotNil(gotUser)
}

func TestLogin_BasicAuthInvalidPassword(t *testing.T) {
	t.Parallel()
	uow := mocks.NewUnitOfWork(t)
	userRepo := mocks.NewUserRepository(t)
	assert := assert.New(t)
	require := require.New(t)
	logger := slog.Default()

	// Mock the repository to return a user
	// The BasicAuthStrategy will still reject due to wrong password
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(userRepo, nil).Once()
	userRepo.EXPECT().GetByUsername(context.Background(), "user").Return(&dto.UserRead{
		ID:       uuid.New(),
		Username: "user",
		Email:    "user@example.com",
	}, nil).Once()

	s := authsvc.NewWithBasic(uow, logger)
	gotUser, err := s.Login(context.Background(), "user", "wrongpassword")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthUoWFactoryError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewStrategy(t)
	expectedErr := errors.New("uow error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(nil, expectedErr).Once()
	s := authsvc.New(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthUserNotFound(t *testing.T) {
	t.Parallel()
	uow := mocks.NewUnitOfWork(t)
	userRepo := mocks.NewUserRepository(t)
	assert := assert.New(t)
	require := require.New(t)
	logger := slog.Default()

	// Test with username
	t.Run("with username", func(t *testing.T) {
		// Mock the repository to return nil user (not found)
		uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(userRepo, nil).Once()
		userRepo.EXPECT().GetByUsername(context.Background(), "notfound").Return(nil, nil).Once()

		s := authsvc.NewWithBasic(uow, logger)
		gotUser, err := s.Login(context.Background(), "notfound", "password")
		require.Error(err)
		assert.Nil(gotUser)
	})

	// Test with email
	t.Run("with email", func(t *testing.T) {
		// Mock the repository to return nil user (not found)
		uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(userRepo, nil).Once()
		userRepo.EXPECT().GetByEmail(
			context.Background(),
			"notfound@example.com",
		).Return(nil, nil).Once()

		s := authsvc.NewWithBasic(uow, logger)
		gotUser, err := s.Login(context.Background(), "notfound@example.com", "password")
		require.Error(err)
		assert.Nil(gotUser)
	})
}

func TestLogin_RepoErrorWithUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(nil, expectedErr).Once()
	s := authsvc.New(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.Error(err)
	assert.Nil(gotUser)
}

func TestLogin_BasicAuthRepoErrorWithUser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	authStrategy := mocks.NewStrategy(t)
	expectedErr := errors.New("db error")
	authStrategy.EXPECT().Login(
		mock.Anything,
		"user",
		"password").Return(nil, expectedErr).Once()
	s := authsvc.New(nil, authStrategy, slog.Default())
	gotUser, err := s.Login(context.Background(), "user", "password")
	require.Error(err)
	assert.Nil(gotUser)
}
