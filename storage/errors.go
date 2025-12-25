package storage

import "errors"

var (
	// ErrFileNotFound is returned when a file doesn't exist
	ErrFileNotFound = errors.New("storage: file not found")

	// ErrInvalidPath is returned when a path is invalid
	ErrInvalidPath = errors.New("storage: invalid path")

	// ErrPermissionDenied is returned when access is denied
	ErrPermissionDenied = errors.New("storage: permission denied")

	// ErrDiskNotFound is returned when a disk is not registered
	ErrDiskNotFound = errors.New("storage: disk not found")

	// ErrInvalidSignature is returned when a signed URL signature is invalid
	ErrInvalidSignature = errors.New("storage: invalid signature")

	// ErrURLExpired is returned when a signed URL has expired
	ErrURLExpired = errors.New("storage: URL expired")

	// ErrNotImplemented is returned when a feature is not supported
	ErrNotImplemented = errors.New("storage: not implemented")
)
