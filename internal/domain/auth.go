package domain

import "errors"

var ErrTokenExpired = errors.New("token expired")

type AccessTokenClaims struct {
	AccountID string `json:"account_id"`
}

type AccessTokenProvider interface {
	NewAccessToken(claims *AccessTokenClaims) (string, error)

	// ParseAccessToken parses, validates, verifies the signature, and returns the
	// parsed token claims. Returns domain.ErrTokenExpired if the token is expired.
	ParseAccessToken(tokenStr string) (*AccessTokenClaims, error)
}
