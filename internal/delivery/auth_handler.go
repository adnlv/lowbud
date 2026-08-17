package delivery

import (
	"github.com/adnlv/lowbud/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	DB                  *pgxpool.Pool
	UUIDProvider        domain.UUIDProvider
	PasswordHasher      domain.PasswordHasher
	AccessTokenProvider domain.AccessTokenProvider
}

func NewAuthHandler(
	db *pgxpool.Pool,
	uuidProvider domain.UUIDProvider,
	passwordHasher domain.PasswordHasher,
	accessTokenProvider domain.AccessTokenProvider,
) *AuthHandler {
	return &AuthHandler{
		DB:                  db,
		UUIDProvider:        uuidProvider,
		PasswordHasher:      passwordHasher,
		AccessTokenProvider: accessTokenProvider,
	}
}
