package auth

import "github.com/google/uuid"

type AccessTokenPayload struct {
	AccountID uuid.UUID `json:"account_id"`
}

type AccessTokenManager interface {
	Generate(payload *AccessTokenPayload) (string, error)
	Validate(tokenStr string) (*AccessTokenPayload, error)
}
