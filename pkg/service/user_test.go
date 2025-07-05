package service

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper to create a service with mocks
func newUserServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (*UserService, *fixtures.MockUserRepository, *fixtures.MockUnitOfWork) {
	userRepo := fixtures.NewMockUserRepository(t)
	uow := fixtures.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(userRepo)
	svc := NewUserService(func() (repository.UnitOfWork, error) { return uow, nil })
	return svc, userRepo, uow
}

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil)

	userRepo.On("Create", mock.Anything).Return(nil)

	u, err := svc.CreateUser("alice", "alice@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
}

func TestCreateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	userRepo.On("Create", mock.Anything).Return(errors.New("db error"))

	u, err := svc.CreateUser("bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestCreateUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	u, err := svc.CreateUser("bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestCreateUser_CommitError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(errors.New("commit error")).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	userRepo.On("Create", mock.Anything).Return(nil)

	u, err := svc.CreateUser("bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Get", user.ID).Return(user, nil)

	got, err := svc.GetUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUser_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	userRepo.On("Get", mock.Anything).Return(&domain.User{}, domain.ErrAccountNotFound)

	got, err := svc.GetUser(uuid.New())
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	_, err := svc.GetUser(uuid.New())
	assert.Error(t, err)
}

func TestGetUserByEmail_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)

	user := &domain.User{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.On("GetByEmail", user.Email).Return(user, nil)

	got, err := svc.GetUserByEmail(user.Email)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	userRepo.On("GetByEmail", mock.Anything).Return((*domain.User)(nil), errors.New("not found"))

	got, err := svc.GetUserByEmail("notfound@example.com")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByEmail_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	_, err := svc.GetUserByEmail("notfound@example.com")
	assert.Error(t, err)
}

func TestGetUserByUsername_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("GetByUsername", user.Username).Return(user, nil)

	got, err := svc.GetUserByUsername(user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	userRepo.On("GetByUsername", mock.Anything).Return((*domain.User)(nil), errors.New("not found"))

	got, err := svc.GetUserByUsername("notfound")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByUsername_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	_, err := svc.GetUserByUsername("notfound")
	assert.Error(t, err)
}

func TestUpdateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Update", user).Return(nil)

	err := svc.UpdateUser(user)
	assert.NoError(t, err)
}

func TestUpdateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Update", user).Return(errors.New("db error"))

	err := svc.UpdateUser(user)
	assert.Error(t, err)
}

func TestUpdateUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	err := svc.UpdateUser(&domain.User{})
	assert.Error(t, err)
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Commit().Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo)
	id := uuid.New()
	userRepo.On("Delete", id).Return(nil)

	err := svc.DeleteUser(id)
	assert.NoError(t, err)
}

func TestDeleteUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().Begin().Return(nil).Once()
	uow.EXPECT().Rollback().Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo)
	id := uuid.New()
	userRepo.On("Delete", id).Return(errors.New("db error"))

	err := svc.DeleteUser(id)
	assert.Error(t, err)
}

func TestDeleteUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	err := svc.DeleteUser(uuid.New())
	assert.Error(t, err)
}

func TestValidUser_True(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.On("Valid", id, "password").Return(true)

	ok := svc.ValidUser(id, "password")
	assert.True(t, ok)
}

func TestValidUser_False(t *testing.T) {
	t.Parallel()
	svc, userRepo, _ := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.On("Valid", id, "wrongpass").Return(false)

	ok := svc.ValidUser(id, "wrongpass")
	assert.False(t, ok)
}

func TestValidUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(func() (repository.UnitOfWork, error) { return nil, errors.New("uow error") })
	ok := svc.ValidUser(uuid.New(), "password")
	assert.False(t, ok)
}
