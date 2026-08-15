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
	Secret               []byte
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

func NewJwtManager(secret string, accessTokenDuration, refreshTokenDuration time.Duration) *JwtManager {
	return &JwtManager{
		Secret:               []byte(secret),
		AccessTokenDuration:  accessTokenDuration,
		RefreshTokenDuration: refreshTokenDuration,
	}
}

func (m *JwtManager) GenerateAccessToken(payload *AccessTokenPayload) (string, error) {
	claims := JwtClaims{
		AccessTokenPayload: *payload,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "lowbud-api",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.Secret)
	if err != nil {
		return "", fmt.Errorf("JwtManager.GenerateAccessToken: failed to sign token: %v", err)
	}
	return signedToken, nil
}

func (m *JwtManager) ParseAccessToken(tokenStr string) (*AccessTokenPayload, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("JwtManager.ParseAccessToken: unexpected signing method: %v", token.Header["alg"])
		}
		return m.Secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("JwtManager.Validate: token has expired")
		}
		return nil, fmt.Errorf("JwtManager.ParseAccessToken: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("JwtManager.Validate: invalid token claims")
	}
	return &claims.AccessTokenPayload, nil
}

func (m *JwtManager) GenerateRefreshToken() (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "lowbud-api",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.RefreshTokenDuration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.Secret)
	if err != nil {
		return "", fmt.Errorf("JwtManager.GenerateRefreshToken: failed to sign token: %v", err)
	}
	return signedToken, nil
}

func (m *JwtManager) ValidateRefreshToken(tokenStr string) error {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("JwtManager.ValidateRefreshToken: unexpected signing method: %v", token.Header["alg"])
		}
		return m.Secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return fmt.Errorf("JwtManager.ValidateRefreshToken: token has expired")
		}
		return fmt.Errorf("JwtManager.ValidateRefreshToken: invalid token: %w", err)
	}

	_, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return fmt.Errorf("JwtManager.ValidateRefreshToken: invalid token claims")
	}
	return nil
}
