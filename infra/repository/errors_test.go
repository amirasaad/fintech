package repository

import (
	"errors"
	"testing"

	"github.com/amirasaad/fintech/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMapGormErrorToDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    error
		expected error
	}{
		{
			name:     "nil error returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "duplicate key error maps to ErrAlreadyExists",
			input:    gorm.ErrDuplicatedKey,
			expected: domain.ErrAlreadyExists,
		},
		{
			name:     "record not found error maps to ErrNotFound",
			input:    gorm.ErrRecordNotFound,
			expected: domain.ErrNotFound,
		},
		{
			name:     "non-GORM error returns original",
			input:    errors.New("some other error"),
			expected: nil, // We'll check the message directly
		},
		{
			name:     "wrapped duplicate key error maps correctly",
			input:    errors.Join(errors.New("outer error"), gorm.ErrDuplicatedKey),
			expected: domain.ErrAlreadyExists,
		},
		{
			name:     "wrapped record not found error maps correctly",
			input:    errors.Join(errors.New("outer error"), gorm.ErrRecordNotFound),
			expected: domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := MapGormErrorToDomain(tt.input)

			switch {
			case tt.name == "non-GORM error returns original":
				// For non-GORM errors, verify original error is returned
				require.Error(t, result)
				assert.Equal(t, tt.input.Error(), result.Error())
			case tt.expected == nil:
				require.NoError(t, result)
			default:
				require.Error(t, result)
				assert.ErrorIs(t, result, tt.expected)
			}
		})
	}
}

func TestMapGormErrorToDomain_ErrorChainTraversal(t *testing.T) {
	t.Parallel()

	t.Run("finds GORM error deep in chain", func(t *testing.T) {
		t.Parallel()
		// Create a deeply wrapped error chain
		innerErr := gorm.ErrDuplicatedKey
		middleErr := errors.Join(errors.New("middle"), innerErr)
		outerErr := errors.Join(errors.New("outer"), middleErr)

		result := MapGormErrorToDomain(outerErr)

		require.Error(t, result)
		assert.ErrorIs(t, result, domain.ErrAlreadyExists)
	})

	t.Run("handles multiple GORM errors in chain", func(t *testing.T) {
		t.Parallel()
		// When multiple GORM errors exist, first one found should be returned
		err := errors.Join(gorm.ErrRecordNotFound, gorm.ErrDuplicatedKey)

		result := MapGormErrorToDomain(err)

		require.Error(t, result)
		// Should map to the first error found (order may vary, but one should match)
		assert.True(t,
			errors.Is(result, domain.ErrNotFound) || errors.Is(result, domain.ErrAlreadyExists),
			"should map to either ErrNotFound or ErrAlreadyExists",
		)
	})
}

func TestWrapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       func() error
		expected error
	}{
		{
			name: "wraps nil error",
			op: func() error {
				return nil
			},
			expected: nil,
		},
		{
			name: "wraps duplicate key error",
			op: func() error {
				return gorm.ErrDuplicatedKey
			},
			expected: domain.ErrAlreadyExists,
		},
		{
			name: "wraps record not found error",
			op: func() error {
				return gorm.ErrRecordNotFound
			},
			expected: domain.ErrNotFound,
		},
		{
			name: "wraps non-GORM error",
			op: func() error {
				return errors.New("custom error")
			},
			expected: nil, // We'll check the message directly
		},
		{
			name: "wraps wrapped GORM error",
			op: func() error {
				return errors.Join(errors.New("wrapper"), gorm.ErrDuplicatedKey)
			},
			expected: domain.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := WrapError(tt.op)

			switch {
			case tt.name == "wraps non-GORM error":
				// For non-GORM errors, verify original error is returned
				require.Error(t, result)
				assert.Equal(t, "custom error", result.Error())
			case tt.expected == nil:
				require.NoError(t, result)
			default:
				require.Error(t, result)
				assert.ErrorIs(t, result, tt.expected)
			}
		})
	}
}

func TestWrapError_Panics(t *testing.T) {
	t.Parallel()

	t.Run("handles panic in operation", func(t *testing.T) {
		t.Parallel()
		// This test ensures WrapError doesn't catch panics
		// (it shouldn't - panics should propagate)
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to propagate")
			}
		}()

		_ = WrapError(func() error {
			panic("test panic")
		})
	})
}

func TestMapGormErrorToDomain_UnmappedGormErrors(t *testing.T) {
	t.Parallel()

	t.Run("unmapped GORM errors return original", func(t *testing.T) {
		t.Parallel()
		// Test with a GORM error that we haven't mapped yet
		// (if such errors exist in future GORM versions)
		customGormErr := errors.New("unmapped gorm error")

		result := MapGormErrorToDomain(customGormErr)

		require.Error(t, result)
		assert.Equal(t, customGormErr, result)
	})
}
