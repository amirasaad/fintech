package user_test

import (
	"context"
	"errors"
	"testing"

	"log/slog"

	"github.com/amirasaad/fintech/pkg/domain/user"
	"github.com/amirasaad/fintech/pkg/dto"

	"github.com/amirasaad/fintech/internal/fixtures/mocks"
	"github.com/amirasaad/fintech/pkg/repository"
	userrepo "github.com/amirasaad/fintech/pkg/repository/user"
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
}) (*usersvc.Service, *mocks.UserRepository, *mocks.UnitOfWork) {
	userRepo := mocks.NewUserRepository(t)
	uow := mocks.NewUnitOfWork(t)

	// Match any repository type that is assignable to *userrepo.Repository
	uow.EXPECT().GetRepository(mock.MatchedBy(func(repoType any) bool {
		// Check if the type is a pointer to userrepo.Repository
		_, ok1 := repoType.(*userrepo.Repository)
		// Also match nil interface type (common in tests)
		_, ok2 := repoType.(interface{ IsA() })
		return ok1 || ok2
	})).Return(userRepo, nil).Maybe()

	// Setup the Do method to execute the callback with the mock UoW
	uow.EXPECT().Do(mock.Anything, mock.AnythingOfType("func(repository.UnitOfWork) error")).
		Return(nil).
		RunAndReturn(func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		}).
		Maybe()

	svc := usersvc.New(uow, slog.Default())
	return svc, userRepo, uow
}

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	u, err := svc.CreateUser(context.Background(), "alice", "alice@example.com", "password")
	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "alice", u.Username)
}

func TestCreateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(userRepo, nil).Maybe()
	userRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db error"))
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
	user := &dto.UserRead{ID: uuid.New(), Username: "alice"}
	userRepo.EXPECT().Get(mock.Anything, user.ID).Return(user, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUser(context.Background(), user.ID.String())
	require.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUser_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, user.ErrUserNotFound)
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
	uow := mocks.NewUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.New(uow, slog.Default())
	_, err := svc.GetUser(context.Background(), uuid.New().String())
	require.Error(t, err)
}

func TestGetUserByEmail_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)

	user := &dto.UserRead{ID: uuid.New(), Email: "alice@example.com"}
	userRepo.EXPECT().GetByEmail(mock.Anything, user.Email).Return(user, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(nil, user.ErrUserNotFound)
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
	uow := mocks.NewUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.New(uow, slog.Default())
	_, err := svc.GetUserByEmail(context.Background(), "user@example.com")
	require.Error(t, err)
}

func TestGetUserByUsername_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	u := &dto.UserRead{ID: uuid.New(), Username: "alice"}
	userRepo.EXPECT().GetByUsername(mock.Anything, u.Username).Return(u, nil)
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	got, err := svc.GetUserByUsername(context.Background(), u.Username)
	require.NoError(t, err)
	assert.Equal(t, u, got)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userRepo.EXPECT().GetByUsername(
		mock.Anything,
		mock.Anything,
	).Return(
		nil,
		user.ErrUserNotFound,
	)
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
	uow := mocks.NewUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.New(uow, slog.Default())
	_, err := svc.GetUserByUsername(context.Background(), "username")
	require.Error(t, err)
}

func TestUpdateUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	user := &dto.UserRead{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
	uu := &dto.UserUpdate{
		Username: &user.Username,
		Email:    &user.Email,
	}

	uow.EXPECT().Do(
		mock.Anything,
		mock.Anything,
	).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	// Mock the Get call that happens before Update
	userRepo.EXPECT().Get(mock.Anything, user.ID).Return(user, nil)
	// Mock the Update call with the correct parameter types
	userRepo.EXPECT().Update(mock.Anything, user.ID, uu).Return(nil)

	err := svc.UpdateUser(context.Background(), user.ID.String(), uu)
	require.NoError(t, err)
}

func TestUpdateUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	userID := uuid.New()
	ur := &dto.UserRead{ID: userID, Username: "oldusername", Email: "old@example.com"}
	newUsername := "alice"
	names := "alice,bob"
	uu := &dto.UserUpdate{Username: &newUsername, Names: &names}

	uow.EXPECT().Do(
		mock.Anything,
		mock.Anything,
	).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	// Mock the Get call that happens before Update
	userRepo.EXPECT().Get(mock.Anything, userID).Return(ur, nil)
	// Mock the Update call with the correct parameter types
	userRepo.EXPECT().Update(mock.Anything, userID, uu).Return(errors.New("db error"))

	err := svc.UpdateUser(context.Background(), userID.String(), uu)
	require.Error(t, err)
}

func TestUpdateUser_UoWFactoryError(t *testing.T) {
	uow := mocks.NewUnitOfWork(t)
	expectedErr := errors.New("factory error")
	uow.EXPECT().Do(mock.Anything, mock.Anything).Return(expectedErr).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return expectedErr
		},
	)

	svc := usersvc.New(uow, slog.Default())
	err := svc.UpdateUser(context.Background(), uuid.New().String(), nil)
	require.Error(t, err)
}

func TestUpdateUser_CallsGetRepositoryOnce(t *testing.T) {
	t.Parallel()
	uow := mocks.NewUnitOfWork(t)
	userRepo := mocks.NewUserRepository(t)

	uow.EXPECT().Do(
		mock.Anything,
		mock.Anything,
	).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	// We expect GetRepository to be called exactly once for both Get and Update operations
	uow.EXPECT().GetRepository((*userrepo.Repository)(nil)).Return(userRepo, nil).Once()

	userID := uuid.New()
	user := &dto.UserRead{ID: userID, Username: "oldusername", Email: "old@example.com"}
	newUsername := "alice"
	uu := &dto.UserUpdate{Username: &newUsername}

	// Mock the Get call that happens before Update
	userRepo.EXPECT().Get(mock.Anything, userID).Return(user, nil)
	// Mock the Update call with the correct parameter types (value, not pointer)
	userRepo.EXPECT().Update(mock.Anything, userID, uu).Return(nil)

	svc := usersvc.New(uow, slog.Default())
	err := svc.UpdateUser(
		context.Background(),
		userID.String(),
		uu,
	)
	require.NoError(t, err)
}

func TestDeleteUser_Success(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Delete(context.Background(), id).Return(nil)
	uow.EXPECT().Do(
		mock.Anything,
		mock.Anything,
	).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	err := svc.DeleteUser(context.Background(), id.String())
	require.NoError(t, err)
}

func TestDeleteUser_RepoError(t *testing.T) {
	t.Parallel()
	svc, userRepo, uow := newUserServiceWithMocks(t)
	id := uuid.New()
	userRepo.EXPECT().Delete(context.Background(), id).Return(errors.New("db error"))
	uow.EXPECT().Do(
		mock.Anything,
		mock.Anything,
	).Return(nil).RunAndReturn(
		func(ctx context.Context, fn func(repository.UnitOfWork) error) error {
			return fn(uow)
		},
	)

	err := svc.DeleteUser(context.Background(), id.String())
	require.Error(t, err)
}
