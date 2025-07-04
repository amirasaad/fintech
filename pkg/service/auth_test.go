package service

import (
	"errors"
	"os"
	"testing"

	fixtures "github.com/amirasaad/fintech/internal/fixtures/repository"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestCheckPasswordHash(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	s := &AuthService{}
	if !s.CheckPasswordHash("password", string(hash)) {
		t.Error("expected password to match hash")
	}
	if s.CheckPasswordHash("wrong", string(hash)) {
		t.Error("expected wrong password to not match hash")
	}
}

func TestValidEmail(t *testing.T) {
	s := &AuthService{}
	if !s.ValidEmail("fixtures@example.com") {
		t.Error("expected valid email")
	}
	if s.ValidEmail("not-an-email") {
		t.Error("expected invalid email")
	}
}

func TestLogin_Success(t *testing.T) {
	os.Setenv("JWT_SECRET_KEY", "testsecret") // nolint: errcheck
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "password")
	if err != nil || gotUser == nil || token == "" {
		t.Errorf("expected login success, got err=%v user=%v token=%v", err, gotUser, token)
	}
	// Validate token
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) { return []byte("testsecret"), nil })
	if err != nil || !parsed.Valid {
		t.Error("expected valid jwt token")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "wrong")
	if err != nil || gotUser != nil || token != "" {
		t.Errorf("expected login fail, got err=%v user=%v token=%v", err, gotUser, token)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	assert := assert.New(t)
	repo := fixtures.NewMockUserRepository(t)

	repo.EXPECT().GetByEmail("notfound@example.com").Return(nil, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("notfound@example.com", "password")
	assert.Nil(gotUser)
	assert.Empty(token)
	assert.NoError(err)
}

func TestLogin_UnitOfWorkError(t *testing.T) {
	s := NewAuthService(func() (repository.UnitOfWork, error) { return nil, errors.New("fail") })
	gotUser, token, err := s.Login("user@example.com", "password")
	if err == nil || gotUser != nil || token != "" {
		t.Errorf("expected error, got err=%v user=%v token=%v", err, gotUser, token)
	}
}

func TestLogin_RepoError(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	repo := fixtures.NewMockUserRepository(t)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}

	repo.EXPECT().GetByEmail("user@example.com").Return(user, errors.New("db error")).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Times(2)

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)

	repo.EXPECT().GetByUsername("user").Return(user, errors.New("db error")).Once()
	gotUser, token, err = s.Login("user", "password")
	assert.Error(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

func TestLogin_JWTSignError(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	os.Setenv("JWT_SECRET_KEY", "") // nolint: errcheck
	gotUser, token, err := s.Login("user@example.com", "password")
	if err == nil || gotUser != nil || token != "" {
		t.Errorf("expected JWT sign error, got err=%v user=%v token=%v", err, gotUser, token)
	}
}

func TestJWTAuthStrategy_Login_GetByEmailError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, errors.New("db error")).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

func TestGetCurrentUserId_InvalidToken(t *testing.T) {
	s := &AuthService{}
	token := &jwt.Token{}
	_, err := s.GetCurrentUserId(token)
	assert.Error(t, err)
}

func TestGetCurrentUserId_MissingClaim(t *testing.T) {
	s := &AuthService{}
	token := jwt.New(jwt.SigningMethodHS256)
	_, err := s.GetCurrentUserId(token)
	assert.Error(t, err)
}

func TestBasicAuthStrategy_Login_Success(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByUsername("user").Return(user, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()
	s := NewBasicAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user", "password")
	assert.NoError(t, err)
	assert.NotNil(t, gotUser)
	assert.Empty(t, token)
}

func TestBasicAuthStrategy_Login_InvalidPassword(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByUsername("user").Return(user, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()
	s := NewBasicAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user", "wrong")
	assert.NoError(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

func TestBasicAuthStrategy_Login_UoWFactoryError(t *testing.T) {
	s := NewBasicAuthService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	gotUser, token, err := s.Login("user", "password")
	assert.Error(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

func TestBasicAuthStrategy_Login_UserNotFound(t *testing.T) {
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByUsername("notfound").Return(nil, nil).Once()
	repo.EXPECT().GetByEmail("notfound@example.com").Return(nil, nil).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Times(2)
	s := NewBasicAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("notfound", "password")
	assert.NoError(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)

	gotUser, token, err = s.Login("notfound@example.com", "password")
	assert.NoError(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

func TestJWTAuthStrategy_Login_RepoErrorWithUser(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByUsername("user").Return(user, errors.New("db error")).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user", "password")
	assert.Error(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

func TestBasicAuthStrategy_Login_RepoErrorWithUser(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := fixtures.NewMockUserRepository(t)
	repo.EXPECT().GetByUsername("user").Return(user, errors.New("db error")).Once()
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()
	s := NewBasicAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user", "password")
	assert.Error(t, err)
	assert.Nil(t, gotUser)
	assert.Empty(t, token)
}

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
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	os.Setenv("JWT_SECRET_KEY", "testsecret") // nolint: errcheck
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = s.Login("user@example.com", "password")
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
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	os.Setenv("JWT_SECRET_KEY", "testsecret") // nolint: errcheck
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = s.Login("user@example.com", "wrong")
	}
}

func BenchmarkLogin_UserNotFound(b *testing.B) {
	repo := fixtures.NewMockUserRepository(b)
	repo.EXPECT().GetByEmail("notfound@example.com").Return(&domain.User{}, errors.New("user not found")).Maybe()
	uow := fixtures.NewMockUnitOfWork(b)
	uow.EXPECT().UserRepository().Return(repo).Maybe()
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = s.Login("notfound@example.com", "password")
	}
}
