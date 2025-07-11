package service

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Helper to create a service with mocks
func newUserServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (*UserService, *fixtures.MockUserRepository, *fixtures.MockUnitOfWork) {
	userRepo := fixtures.NewMockUserRepository(t)
	uow := fixtures.NewMockUnitOfWork(t)
	svc := NewUserService(uow, slog.Default())
	return svc, userRepo, uow
}

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Create(mock.Anything).Return(nil)
	uow.EXPECT().Commit().Return(nil)

	u, err := svc.CreateUser(context.Background(), "alice", "alice@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
}

func TestCreateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))
	uow.EXPECT().Commit().Return(nil)

	u, err := svc.CreateUser(context.Background(), "bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestCreateUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	u, err := svc.CreateUser(context.Background(), "bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestCreateUser_CommitError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Create(mock.Anything).Return(nil)
	uow.EXPECT().Commit().Return(errors.New("commit error"))

	u, err := svc.CreateUser(context.Background(), "bob", "bob@example.com", "password")
	assert.Error(t, err)
	assert.Nil(t, u)
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Get", user.ID).Return(user, nil)
	uow.EXPECT().Commit().Return(nil)

	got, err := svc.GetUser(user.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUser_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Get(mock.Anything).Return(&domain.User{}, domain.ErrAccountNotFound)
	uow.EXPECT().Commit().Return(nil)

	got, err := svc.GetUser(uuid.New().String())
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	_, err := svc.GetUser(uuid.New().String())
	assert.Error(t, err)
}

func TestGetUserByEmail_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)

	user := &domain.User{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.EXPECT().GetByEmail(user.Email).Return(user, nil)
	uow.EXPECT().Commit().Return(nil)

	got, err := svc.GetUserByEmail(user.Email)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().GetByEmail(mock.Anything).Return((*domain.User)(nil), errors.New("not found"))
	uow.EXPECT().Commit().Return(nil)

	got, err := svc.GetUserByEmail("notfound@example.com")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByEmail_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	_, err := svc.GetUserByEmail("notfound@example.com")
	assert.Error(t, err)
}

func TestGetUserByUsername_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.EXPECT().GetByUsername(user.Username).Return(user, nil)
	uow.EXPECT().Commit().Return(nil)

	got, err := svc.GetUserByUsername(user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().GetByUsername(mock.Anything).Return((*domain.User)(nil), errors.New("not found"))
	uow.EXPECT().Commit().Return(nil)

	got, err := svc.GetUserByUsername("notfound")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByUsername_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	_, err := svc.GetUserByUsername("notfound")
	assert.Error(t, err)
}

func TestUpdateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	uuidStr := user.ID.String()
	userRepo.EXPECT().Get(user.ID).Return(user, nil)
	userRepo.EXPECT().Update(user).Return(nil)
	uow.EXPECT().Commit().Return(nil)

	err := svc.UpdateUser(uuidStr, func(u *domain.User) error {
		u.Username = "updated"
		return nil
	})
	assert.NoError(t, err)
}

func TestUpdateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	uuidStr := user.ID.String()
	userRepo.EXPECT().Get(user.ID).Return(user, nil)
	userRepo.EXPECT().Update(user).Return(errors.New("db error"))
	uow.EXPECT().Commit().Return(nil)

	err := svc.UpdateUser(uuidStr, func(u *domain.User) error {
		u.Username = "updated"
		return nil
	})
	assert.Error(t, err)
}

func TestUpdateUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	err := svc.UpdateUser(uuid.New().String(), func(u *domain.User) error { return nil })
	assert.Error(t, err)
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Delete(id).Return(nil)
	uow.EXPECT().Commit().Return(nil)

	err := svc.DeleteUser(id.String())
	assert.NoError(t, err)
}

func TestDeleteUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Delete(id).Return(errors.New("db error"))
	uow.EXPECT().Commit().Return(nil)

	err := svc.DeleteUser(id.String())
	assert.Error(t, err)
}

func TestDeleteUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	err := svc.DeleteUser(uuid.New().String())
	assert.Error(t, err)
}

func TestValidUser_True(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Valid(id, "password").Return(true)
	uow.EXPECT().Commit().Return(nil)

	ok, _ := svc.ValidUser(id.String(), "password")
	assert.True(t, ok)
}

func TestValidUser_False(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Valid(id, "wrongpass").Return(false)
	uow.EXPECT().Commit().Return(nil)

	ok, _ := svc.ValidUser(id.String(), "wrongpass")
	assert.False(t, ok)
}

func TestValidUser_UoWFactoryError(t *testing.T) {
	t.Parallel()
	svc := NewUserService(&fixtures.MockUnitOfWork{}, slog.Default())
	ok, _ := svc.ValidUser(uuid.New().String(), "password")
	assert.False(t, ok)
}
