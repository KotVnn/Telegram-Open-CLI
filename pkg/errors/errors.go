package errors

import "errors"

var (
	// ErrSessionNotFound is returned when a session doesn't exist.
	ErrSessionNotFound = errors.New("session not found")

	// ErrProjectNotFound is returned when a project doesn't exist.
	ErrProjectNotFound = errors.New("project not found")

	// ErrUserNotFound is returned when a user doesn't exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrUnauthorized is returned when a user is not authorized.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when a user doesn't have permission.
	ErrForbidden = errors.New("forbidden")

	// ErrBackendUnavailable is returned when a backend is not available.
	ErrBackendUnavailable = errors.New("backend unavailable")

	// ErrBackendTimeout is returned when a backend operation times out.
	ErrBackendTimeout = errors.New("backend timeout")

	// ErrRateLimited is returned when rate limit is exceeded.
	ErrRateLimited = errors.New("rate limited")

	// ErrInvalidConfig is returned when configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrStorageUnavailable is returned when storage is not available.
	ErrStorageUnavailable = errors.New("storage unavailable")

	// ErrPluginLoadFailed is returned when a plugin fails to load.
	ErrPluginLoadFailed = errors.New("plugin load failed")
)
