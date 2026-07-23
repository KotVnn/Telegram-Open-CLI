package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorVariablesExist(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrSessionNotFound", ErrSessionNotFound, "session not found"},
		{"ErrProjectNotFound", ErrProjectNotFound, "project not found"},
		{"ErrUserNotFound", ErrUserNotFound, "user not found"},
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized"},
		{"ErrForbidden", ErrForbidden, "forbidden"},
		{"ErrBackendUnavailable", ErrBackendUnavailable, "backend unavailable"},
		{"ErrBackendTimeout", ErrBackendTimeout, "backend timeout"},
		{"ErrRateLimited", ErrRateLimited, "rate limited"},
		{"ErrInvalidConfig", ErrInvalidConfig, "invalid configuration"},
		{"ErrStorageUnavailable", ErrStorageUnavailable, "storage unavailable"},
		{"ErrPluginLoadFailed", ErrPluginLoadFailed, "plugin load failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.EqualError(t, tt.err, tt.expected)
		})
	}
}

func TestErrorVariablesAreDistinct(t *testing.T) {
	errors := []error{
		ErrSessionNotFound,
		ErrProjectNotFound,
		ErrUserNotFound,
		ErrUnauthorized,
		ErrForbidden,
		ErrBackendUnavailable,
		ErrBackendTimeout,
		ErrRateLimited,
		ErrInvalidConfig,
		ErrStorageUnavailable,
		ErrPluginLoadFailed,
	}

	seen := make(map[error]bool)
	for _, err := range errors {
		assert.False(t, seen[err], "duplicate error variable: %v", err)
		seen[err] = true
	}
}
