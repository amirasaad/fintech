package service

import (
	"errors"
	"os"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/test"
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
	if !s.ValidEmail("test@example.com") {
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
	repo := test.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := test.NewMockUnitOfWork(t)
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
	repo := test.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := test.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "wrong")
	if err != nil || gotUser != nil || token != "" {
		t.Errorf("expected login fail, got err=%v user=%v token=%v", err, gotUser, token)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	assert := assert.New(t)
	repo := test.NewMockUserRepository(t)

	repo.EXPECT().GetByEmail("notfound@example.com").Return(&domain.User{}, errors.New("user not found")).Once()
	uow := test.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("notfound@example.com", "password")
	assert.Nil(gotUser)
	assert.Empty(token)
	assert.Error(err)
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

	repo := test.NewMockUserRepository(t)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}

	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := test.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "password")
	if err == nil || gotUser != nil || token != "" {
		t.Errorf("expected error, got err=%v user=%v token=%v", err, gotUser, token)
	}
}

func TestLogin_JWTSignError(t *testing.T) {
	t.Parallel()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user := &domain.User{ID: uuid.New(), Username: "user", Email: "user@example.com", Password: string(hash)}
	repo := test.NewMockUserRepository(t)
	repo.EXPECT().GetByEmail("user@example.com").Return(user, nil).Once()
	uow := test.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(repo).Once()

	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	os.Setenv("JWT_SECRET_KEY", "") // nolint: errcheck
	gotUser, token, err := s.Login("user@example.com", "password")
	if err == nil || gotUser != nil || token != "" {
		t.Errorf("expected JWT sign error, got err=%v user=%v token=%v", err, gotUser, token)
	}
}
