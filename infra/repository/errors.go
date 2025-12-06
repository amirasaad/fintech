package repository

import (
	"errors"

	"github.com/amirasaad/fintech/pkg/domain"
	"gorm.io/gorm"
)

// MapGormErrorToDomain converts GORM errors to domain errors.
// This keeps infrastructure concerns (database errors) within the infrastructure layer.
// Traverses the error chain to find GORM errors and maps them to appropriate domain errors.
func MapGormErrorToDomain(err error) error {
	if err == nil {
		return nil
	}

	// Traverse the error chain to find GORM errors
	// GORM wraps database errors, so we check each level
	currentErr := err
	for currentErr != nil {
		switch {
		case errors.Is(currentErr, gorm.ErrDuplicatedKey):
			return domain.ErrAlreadyExists
		case errors.Is(currentErr, gorm.ErrRecordNotFound):
			return domain.ErrNotFound
			// Add more GORM error mappings as needed
			// case errors.Is(currentErr, gorm.ErrForeignKeyViolated):
			//     return domain.ErrInvalidReference
		}

		// Move to the next error in the chain
		currentErr = errors.Unwrap(currentErr)
	}

	// Return original error if no mapping found
	return err
}

// WrapError wraps a GORM operation and automatically maps errors.
// This helper reduces boilerplate in repository methods while keeping code explicit.
//
// Usage:
//
//	err := WrapError(func() error {
//	    return r.db.WithContext(ctx).Create(user).Error
//	})
func WrapError(op func() error) error {
	return MapGormErrorToDomain(op())
}
