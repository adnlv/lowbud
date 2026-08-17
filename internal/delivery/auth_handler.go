package delivery

import (
	"net/http"
	"time"

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

type RegisterAccountRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Forename string `json:"forename" validate:"required"`
	Surname  string `json:"surname" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (h *AuthHandler) RegisterAccount(w http.ResponseWriter, r *http.Request) {
	req := new(RegisterAccountRequest)
	if err := decodeAndValidateJson(r.Body, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	accountId, err := h.UUIDProvider.NewUUID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	passwordHash, err := h.PasswordHasher.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	account := &domain.Account{
		ID:           accountId,
		Forename:     req.Forename,
		Surname:      req.Surname,
		Email:        req.Email,
		PasswordHash: passwordHash,
		RegisteredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	const insertAccountQuery = `
INSERT INTO accounts (id, forename, surname, email, password_hash, registered_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`
	if _, err = h.DB.Exec(
		r.Context(),
		insertAccountQuery,
		account.ID,
		account.Forename,
		account.Surname,
		account.Email,
		account.PasswordHash,
		account.RegisteredAt,
		account.UpdatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeStatusCode(w, http.StatusCreated)
}
