package user_test

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/amirasaad/fintech/pkg/repository"
	usersvc "github.com/amirasaad/fintech/pkg/service/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper to create a service with mocks
func newUserServiceWithMocks(t interface {
	mock.TestingT
	Cleanup(func())
}) (*usersvc.UserService, *mocks.MockUserRepository, *mocks.MockUnitOfWork) {
	userRepo := mocks.NewMockUserRepository(t)
	uow := mocks.NewMockUnitOfWork(t)
	uow.EXPECT().UserRepository().Return(userRepo, nil).Maybe()
	svc := usersvc.NewUserService(uow, slog.Default())
	return svc, userRepo, uow
}

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Create(mock.Anything).Return(nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	u, err := svc.CreateUser(context.Background(), "alice", "alice@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
}

func TestCreateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Create(mock.Anything).Return(errors.New("db error"))
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	u, err := svc.CreateUser(context.Background(), "bob", "bob@example.com", "password")
	require.Error(t, err)
	assert.Nil(t, u)
}

func TestGetUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.On("Get", user.ID).Return(user, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUser(context.Background(), user.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUser_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Get(mock.Anything).Return(&domain.User{}, domain.ErrAccountNotFound)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUser(context.Background(), uuid.New().String())
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUser_UoWFactoryError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.NewUserService(uow, slog.Default())
	_, err := svc.GetUser(context.Background(), uuid.New().String())
	require.Error(t, err)
}

func TestGetUserByEmail_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)

	user := &domain.User{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.EXPECT().GetByEmail(user.Email).Return(user, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUserByEmail(context.Background(), user.Email)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().GetByEmail(mock.Anything).Return((*domain.User)(nil), errors.New("not found"))
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUserByEmail(context.Background(), "notfound@example.com")
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByEmail_UoWFactoryError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.NewUserService(uow, slog.Default())
	_, err := svc.GetUserByEmail(context.Background(), "user@example.com")
	require.Error(t, err)
}

func TestGetUserByUsername_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	userRepo.EXPECT().GetByUsername(user.Username).Return(user, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUserByUsername(context.Background(), user.Username)
	assert.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().GetByUsername(mock.Anything).Return((*domain.User)(nil), errors.New("not found"))
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUserByUsername(context.Background(), "notfound")
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestGetUserByUsername_UoWFactoryError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.NewUserService(uow, slog.Default())
	_, err := svc.GetUserByUsername(context.Background(), "username")
	require.Error(t, err)
}

func TestUpdateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &domain.User{ID: uuid.New(), Username: "alice"}
	uuidStr := user.ID.String()
	userRepo.EXPECT().Get(user.ID).Return(user, nil)
	userRepo.EXPECT().Update(user).Return(nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	err := svc.UpdateUser(context.Background(), uuidStr, func(u *domain.User) error {
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
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	err := svc.UpdateUser(context.Background(), uuidStr, func(u *domain.User) error {
		u.Username = "updated"
		return nil
	})
	require.Error(t, err)
}

func TestUpdateUser_UoWFactoryError(t *testing.T) {
	uow := mocks.NewMockUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.NewUserService(uow, slog.Default())
	err := svc.UpdateUser(context.Background(), uuid.New().String(), nil)
	require.Error(t, err)
}

func TestUpdateUser_CallsGetRepositoryOnce(t *testing.T) {
	t.Parallel()
	uow := mocks.NewMockUnitOfWork(t)
	userRepo := mocks.NewMockUserRepository(t)
	callCount := 0

	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)
	uow.EXPECT().UserRepository().Return(userRepo, nil).Run(
		func() {
			callCount++
		},
	).Once()

	userID := uuid.New()
	user := &domain.User{ID: userID, Username: "alice"}
	userRepo.EXPECT().Get(userID).Return(user, nil)
	userRepo.EXPECT().Update(user).Return(nil)

	svc := usersvc.NewUserService(uow, slog.Default())
	err := svc.UpdateUser(context.Background(), userID.String(), func(u *domain.User) error {
		u.Username = "updated"
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "GetRepository should be called exactly once")
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Delete(id).Return(nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	err := svc.DeleteUser(context.Background(), id.String())
	assert.NoError(t, err)
}

func TestDeleteUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Delete(id).Return(errors.New("db error"))
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	err := svc.DeleteUser(context.Background(), id.String())
	require.Error(t, err)
}

func TestValidUser_True(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Valid(id, "password").Return(true)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	ok, _ := svc.ValidUser(context.Background(), id.String(), "password")
	assert.True(t, ok)
}

func TestValidUser_False(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Valid(id, "wrongpass").Return(false)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	ok, _ := svc.ValidUser(context.Background(), id.String(), "wrongpass")
	assert.False(t, ok)
}
