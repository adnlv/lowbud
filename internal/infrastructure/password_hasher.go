package infrastructure

import (
	"fmt"

	"github.com/adnlv/lowbud/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type BcryptPasswordHasher struct {
	Cost int
}

func NewBcryptPasswordHasher(cost int) *BcryptPasswordHasher {
	return &BcryptPasswordHasher{Cost: cost}
}

func (h *BcryptPasswordHasher) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	if err != nil {
		return "", fmt.Errorf("generating hash: %w", err)
	}
	return string(bytes), nil
}

func (h *BcryptPasswordHasher) CompareHashAndPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return domain.ErrMismatchedHashAndPassword
	}
	return nil
}
