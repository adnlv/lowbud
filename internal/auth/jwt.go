package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtClaims struct {
	AccessTokenPayload
	jwt.RegisteredClaims
}

type JwtManager struct {
	Secret   []byte
	Duration time.Duration
}

func NewJwtManager(secret string, duration time.Duration) *JwtManager {
	return &JwtManager{
		Secret:   []byte(secret),
		Duration: duration,
	}
}

func (m *JwtManager) Generate(payload *AccessTokenPayload) (string, error) {
	claims := JwtClaims{
		AccessTokenPayload: *payload,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "lowbud-api",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.Duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.Secret)
	if err != nil {
		return "", fmt.Errorf("JwtManager.Generate: failed to sign token: %v", err)
	}
	return signedToken, nil
}

func (m *JwtManager) Validate(tokenStr string) (*AccessTokenPayload, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("JwtManager.Validate: unexpected signing method: %v", token.Header["alg"])
		}
		return m.Secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("JwtManager.Validate: token has expired")
		}
		return nil, fmt.Errorf("JwtManager.Validate: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("JwtManager.Validate: invalid token claims")
	}
	return &claims.AccessTokenPayload, nil
}
