package domain

import "errors"

var ErrInvalidUUID = errors.New("invalid UUID format")

type UUIDProvider interface {
	NewUUID() (string, error)

	// ValidateUUID returns ErrInvalidUUID if s is not a properly formatted UUID.
	ValidateUUID(s string) error
}

var ErrMismatchedHashAndPassword = errors.New("hashedPassword is not the hash of the given password")

type PasswordHasher interface {
	HashPassword(password string) (string, error)

	// CompareHashAndPassword compares a hashed password with its possible plaintext equivalent.
	// Returns nil on success, or ErrMismatchedHashAndPassword on failure.
	CompareHashAndPassword(hashedPassword, password string) error
}
