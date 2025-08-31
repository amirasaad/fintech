package user

import (
	"context"
	"errors"

	"github.com/amirasaad/fintech/pkg/dto"
	"github.com/amirasaad/fintech/pkg/repository/user"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// repository defines the interface for user repository operations
type repository struct {
	// Add repository fields here if needed
	db *gorm.DB
}

func New(db *gorm.DB) user.Repository {
	return &repository{db: db}
}

func (r *repository) GetByEmail(
	ctx context.Context,
	email string,
) (*dto.UserRead, error) {
	var user User
	if err := r.db.WithContext(
		ctx,
	).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return mapModelToDTO(&user), nil
}

func (r *repository) GetByUsername(
	ctx context.Context,
	username string,
) (*dto.UserRead, error) {
	var user User
	if err := r.db.WithContext(
		ctx,
	).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return mapModelToDTO(&user), nil
}

func (r *repository) List(
	ctx context.Context,
	page, pageSize int,
) ([]*dto.UserRead, error) {
	var users []User
	if err := r.db.WithContext(
		ctx,
	).Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, err
	}

	result := make([]*dto.UserRead, 0, len(users))
	for _, user := range users {
		result = append(result, mapModelToDTO(&user))
	}

	return result, nil
}

func (r *repository) Update(
	ctx context.Context,
	id uuid.UUID,
	uu *dto.UserUpdate,
) error {
	updates := make(map[string]interface{})

	// Only include non-nil fields in the update
	if uu.Username != nil {
		updates["username"] = *uu.Username
	}
	if uu.Email != nil {
		updates["email"] = *uu.Email
	}
	if uu.Names != nil {
		updates["names"] = *uu.Names
	}
	if uu.Password != nil {
		updates["password"] = *uu.Password
	}
	if uu.StripeConnectAccountID != nil {
		updates["stripe_connect_account_id"] = *uu.StripeConnectAccountID
	}

	// If no fields to update, return early
	if len(updates) == 0 {
		return nil
	}

	// Update only the specified fields
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *repository) Create(
	ctx context.Context,
	create *dto.UserCreate,
) error {
	user := &User{
		ID:       create.ID,
		Username: create.Username,
		Email:    create.Email,
		Password: create.Password,
		Names:    create.Names,
	}
	return r.db.WithContext(
		ctx,
	).Create(user).Error
}

func (r *repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (*dto.UserRead, error) {
	var user User
	if err := r.db.WithContext(
		ctx,
	).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mapModelToDTO(&user), nil
}

func (r *repository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.db.WithContext(
		ctx,
	).Delete(&User{}, "id = ?", id).Error
}

func (r *repository) Exists(
	ctx context.Context,
	id uuid.UUID,
) (bool, error) {
	var count int64
	err := r.db.WithContext(
		ctx,
	).Model(&User{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	var count int64
	err := r.db.WithContext(
		ctx,
	).Model(&User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) ExistsByUsername(
	ctx context.Context,
	username string,
) (bool, error) {
	var count int64
	err := r.db.WithContext(
		ctx,
	).Model(&User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func mapModelToDTO(user *User) *dto.UserRead {
	return &dto.UserRead{
		ID:                     user.ID,
		Username:               user.Username,
		Email:                  user.Email,
		HashedPassword:         user.Password,
		Names:                  user.Names,
		StripeConnectAccountID: user.StripeConnectAccountID,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
	}
}

var _ user.Repository = (*repository)(nil)
