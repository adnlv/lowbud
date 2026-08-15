package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/adnlv/lowbud/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	model.JwtClaims
}

type JwtManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJwtManager(secret string, ttl time.Duration) *JwtManager {
	return &JwtManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (j *JwtManager) New(accountId uuid.UUID) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "lowbud-api",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		JwtClaims: model.JwtClaims{
			AccountID: accountId,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(j.secret)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func (j *JwtManager) Validate(tokenStr string) (*model.JwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("auth: token has expired")
		}
		return nil, fmt.Errorf("auth: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth: token claims are invalid")
	}
	return &claims.JwtClaims, nil
}
