package domain

import "errors"

// Common domain errors
var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")
	// ErrAlreadyExists is returned when trying to create a resource that already exists
	ErrAlreadyExists = errors.New("resource already exists")
	// ErrValidation is returned when input validation fails
	ErrValidation = errors.New("validation error")
	// ErrUnauthorized is returned when a user is not authorized to perform an action
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned when a user is not allowed to perform an action
	ErrForbidden = errors.New("forbidden")
)
