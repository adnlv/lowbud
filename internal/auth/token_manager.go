package auth

import "github.com/google/uuid"

type AccessTokenPayload struct {
	AccountID uuid.UUID `json:"account_id"`
}

type TokenManager interface {
	GenerateAccessToken(payload *AccessTokenPayload) (string, error)
	ParseAccessToken(tokenStr string) (*AccessTokenPayload, error)
	GenerateRefreshToken() (string, error)
	ValidateRefreshToken(tokenStr string) error
}
