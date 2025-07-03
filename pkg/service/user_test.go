package service

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for repository.UnitOfWork and UserRepository

type MockUserRepo struct {
	mock.Mock
	user *domain.User
}

func (m *MockUserRepo) Create(u *domain.User) error {
	m.user = u
	args := m.Called(u)
	return args.Error(0)
}
func (m *MockUserRepo) Get(id uuid.UUID) (*domain.User, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) GetByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) GetByUsername(username string) (*domain.User, error) {
	args := m.Called(username)
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepo) Update(u *domain.User) error {
	m.user = u
	args := m.Called(u)
	return args.Error(0)
}
func (m *MockUserRepo) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *MockUserRepo) Valid(id uuid.UUID, password string) bool {
	args := m.Called(id, password)
	return args.Bool(0)
}

func (m *MockUoW) UserRepository() repository.UserRepository {
	return m.userRepo
}

// Helper to create a service with mocks
func newUserServiceWithMocks() (*UserService, *MockUserRepo) {
	userRepo := &MockUserRepo{}
	uow := &MockUoW{
		userRepo: userRepo,
	}
	svc := NewUserService(func() (repository.UnitOfWork, error) { return uow, nil })
	return svc, userRepo
}

func TestCreateUser_Success(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	userRepo.On("Create", mock.Anything).Return(nil)

	u, err := svc.CreateUser("alice", "alice@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
}

func TestCreateUser_RepoError(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	userRepo.On("Create", mock.Anything).Return(errors.New("db error"))

	u, err := svc.CreateUser("bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestGetUser_Success(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.user = user
	userRepo.On("Get", user.ID).Return(user, nil)

	got, err := svc.GetUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUser_NotFound(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	userRepo.On("Get", mock.Anything).Return(&domain.User{}, domain.ErrAccountNotFound)

	got, err := svc.GetUser(uuid.New())
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByEmail_Success(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	user := &domain.User{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.On("GetByEmail", user.Email).Return(user, nil)

	got, err := svc.GetUserByEmail(user.Email)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	userRepo.On("GetByEmail", mock.Anything).Return((*domain.User)(nil), errors.New("not found"))

	got, err := svc.GetUserByEmail("notfound@example.com")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByUsername_Success(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("GetByUsername", user.Username).Return(user, nil)

	got, err := svc.GetUserByUsername(user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	userRepo.On("GetByUsername", mock.Anything).Return((*domain.User)(nil), errors.New("not found"))

	got, err := svc.GetUserByUsername("notfound")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestUpdateUser_Success(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Update", user).Return(nil)

	err := svc.UpdateUser(user)
	assert.NoError(t, err)
}

func TestUpdateUser_RepoError(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Update", user).Return(errors.New("db error"))

	err := svc.UpdateUser(user)
	assert.Error(t, err)
}

func TestDeleteUser_Success(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	id := uuid.New()
	userRepo.On("Delete", id).Return(nil)

	err := svc.DeleteUser(id, "password")
	assert.NoError(t, err)
}

func TestDeleteUser_RepoError(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	id := uuid.New()
	userRepo.On("Delete", id).Return(errors.New("db error"))

	err := svc.DeleteUser(id, "password")
	assert.Error(t, err)
}

func TestValidUser_True(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	id := uuid.New()
	userRepo.On("Valid", id, "password").Return(true)

	ok := svc.ValidUser(id, "password")
	assert.True(t, ok)
}

func TestValidUser_False(t *testing.T) {
	svc, userRepo := newUserServiceWithMocks()
	id := uuid.New()
	userRepo.On("Valid", id, "wrongpass").Return(false)

	ok := svc.ValidUser(id, "wrongpass")
	assert.False(t, ok)
}
