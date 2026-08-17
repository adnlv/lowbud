package infrastructure

import (
	"fmt"

	"github.com/adnlv/lowbud/internal/domain"
	"github.com/google/uuid"
)

type GoogleUUIDV7Provider struct{}

func NewGoogleUUIDV7Provider() *GoogleUUIDV7Provider {
	return &GoogleUUIDV7Provider{}
}

func (p *GoogleUUIDV7Provider) NewUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generating UUIDv7: %w", err)
	}
	return id.String(), nil
}

func (p *GoogleUUIDV7Provider) ValidateUUID(s string) error {
	if err := uuid.Validate(s); err != nil {
		return domain.ErrInvalidUUID
	}
	return nil
}
