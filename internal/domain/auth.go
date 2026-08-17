package domain

import "errors"

var (
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrTokenExpired            = errors.New("token expired")
)

type AccessTokenClaims struct {
	AccountID string `json:"account_id"`
}

type AccessTokenProvider interface {
	NewAccessToken(claims *AccessTokenClaims) (string, error)
	ParseAccessToken(tokenStr string) (*AccessTokenClaims, error)
}
