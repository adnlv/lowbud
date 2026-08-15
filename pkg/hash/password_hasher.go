package hash

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password, hash string) error
}

type BcryptPasswordHasher struct {
	Cost int
}

func NewBcryptPasswordHasher(cost int) *BcryptPasswordHasher {
	return &BcryptPasswordHasher{
		Cost: cost,
	}
}

func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	if err != nil {
		return "", fmt.Errorf("BcryptPasswordHasher.Hash: failed to generate from password: %v", err)
	}
	return string(bytes), nil
}

// Compare returns nil on success or an error on failure.
func (h *BcryptPasswordHasher) Compare(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("BcryptPasswordHasher.Compare: invalid password: %v", err)
	}
	return nil
}
