package delivery

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/adnlv/lowbud/internal/auth"
	"github.com/adnlv/lowbud/internal/model"
	"github.com/adnlv/lowbud/pkg/hash"
	"github.com/adnlv/lowbud/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	pgxPool        *pgxpool.Pool
	tokenManager   auth.TokenManager
	passwordHasher hash.PasswordHasher
	uuidGenerator  uuid.Generator
}

func NewAuthHandler(
	pgxPool *pgxpool.Pool,
	tokenManager auth.TokenManager,
	passwordHasher hash.PasswordHasher,
	uuidGenerator uuid.Generator,
) *AuthHandler {
	return &AuthHandler{
		pgxPool:        pgxPool,
		tokenManager:   tokenManager,
		passwordHasher: passwordHasher,
		uuidGenerator:  uuidGenerator,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := decodeAndValidateRequest(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := h.uuidGenerator.Generate()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	passwordHash, err := h.passwordHasher.Hash(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	account := &model.Account{
		ID:           id,
		PasswordHash: passwordHash,
		RegisteredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	tx, err := h.pgxPool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(r.Context())
		}
	}()

	if _, err = tx.Exec(
		r.Context(),
		`INSERT INTO accounts (id, password_hash, registered_at, updated_at) VALUES ($1, $2, $3, $4)`,
		account.ID,
		account.PasswordHash,
		account.RegisteredAt,
		account.UpdatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	accessTokenPayload := &auth.AccessTokenPayload{AccountID: account.ID}
	accessToken, err := h.tokenManager.GenerateAccessToken(accessTokenPayload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	refreshToken, err := h.tokenManager.GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Account:      model.NewAccountView(account),
	})
}
