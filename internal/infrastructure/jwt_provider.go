package infrastructure

import (
	"errors"
	"fmt"
	"time"

	"github.com/adnlv/lowbud/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type JwtClaims struct {
	jwt.RegisteredClaims
	domain.AccessTokenClaims
}

type JwtProvider struct {
	Secret                  []byte
	TokenExpirationDuration time.Duration
}

func NewJwtProvider(secret []byte, tokenExpirationDuration time.Duration) *JwtProvider {
	return &JwtProvider{
		Secret:                  secret,
		TokenExpirationDuration: tokenExpirationDuration,
	}
}

func (p *JwtProvider) NewAccessToken(claims *domain.AccessTokenClaims) (string, error) {
	jwtClaims := JwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "lowbud-api",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.TokenExpirationDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		AccessTokenClaims: *claims,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	signedStr, err := token.SignedString(p.Secret)
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return signedStr, nil
}

func (p *JwtProvider) ParseAccessToken(tokenStr string) (*domain.AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return p.Secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrTokenExpired
		}
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid access token claims")
	}
	return &claims.AccessTokenClaims, nil
}
