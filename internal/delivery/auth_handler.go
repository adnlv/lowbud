package delivery

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/adnlv/lowbud/internal/auth"
	"github.com/adnlv/lowbud/internal/model"
	"github.com/adnlv/lowbud/pkg/hash"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthHandler struct {
	pgxPool            *pgxpool.Pool
	accessTokenManager auth.AccessTokenManager
	passwordHasher     hash.PasswordHasher
}

func NewAuthHandler(pgxPool *pgxpool.Pool, accessTokenManager auth.AccessTokenManager, passwordHasher hash.PasswordHasher) *AuthHandler {
	return &AuthHandler{
		pgxPool:            pgxPool,
		accessTokenManager: accessTokenManager,
		passwordHasher:     passwordHasher,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	passwordHash, err := h.passwordHasher.Hash(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accessTokenPayload := &auth.AccessTokenPayload{AccountID: account.ID}
	accessToken, err := h.accessTokenManager.Generate(accessTokenPayload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(model.AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		Account:     model.NewAccountView(account),
	})
}
