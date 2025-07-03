package service

import (
	"errors"
	"os"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	userByEmail      *domain.User
	userByUsername   *domain.User
	getByEmailErr    error
	getByUsernameErr error
}

func (m *mockUserRepo) Get(id uuid.UUID) (*domain.User, error) { return nil, nil }
func (m *mockUserRepo) GetByEmail(email string) (*domain.User, error) {
	return m.userByEmail, m.getByEmailErr
}
func (m *mockUserRepo) GetByUsername(username string) (*domain.User, error) {
	return m.userByUsername, m.getByUsernameErr
}
func (m *mockUserRepo) Valid(id uuid.UUID, password string) bool { return false }
func (m *mockUserRepo) Create(user *domain.User) error           { return nil }
func (m *mockUserRepo) Update(user *domain.User) error           { return nil }
func (m *mockUserRepo) Delete(id uuid.UUID) error                { return nil }

// mockUOW implements repository.UnitOfWork
// Only UserRepository() is used in AuthService

type mockUOW struct {
	repo repository.UserRepository
}

func (m *mockUOW) Begin() error                                            { return nil }
func (m *mockUOW) Commit() error                                           { return nil }
func (m *mockUOW) Rollback() error                                         { return nil }
func (m *mockUOW) AccountRepository() repository.AccountRepository         { return nil }
func (m *mockUOW) TransactionRepository() repository.TransactionRepository { return nil }
func (m *mockUOW) UserRepository() repository.UserRepository               { return m.repo }

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
	repo := &mockUserRepo{userByEmail: user}
	uow := &mockUOW{repo: repo}
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
	repo := &mockUserRepo{userByEmail: user}
	uow := &mockUOW{repo: repo}
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("user@example.com", "wrong")
	if err != nil || gotUser != nil || token != "" {
		t.Errorf("expected login fail, got err=%v user=%v token=%v", err, gotUser, token)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{userByEmail: nil}
	uow := &mockUOW{repo: repo}
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	gotUser, token, err := s.Login("notfound@example.com", "password")
	if err != nil || gotUser != nil || token != "" {
		t.Errorf("expected login fail, got err=%v user=%v token=%v", err, gotUser, token)
	}
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
	repo := &mockUserRepo{userByEmail: nil, getByEmailErr: errors.New("fail")}
	uow := &mockUOW{repo: repo}
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
	repo := &mockUserRepo{userByEmail: user}
	uow := &mockUOW{repo: repo}
	s := NewAuthService(func() (repository.UnitOfWork, error) { return uow, nil })
	os.Setenv("JWT_SECRET_KEY", "") // nolint: errcheck
	gotUser, token, err := s.Login("user@example.com", "password")
	if err == nil || gotUser != nil || token != "" {
		t.Errorf("expected JWT sign error, got err=%v user=%v token=%v", err, gotUser, token)
	}
}
