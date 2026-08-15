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
	pgxPool        *pgxpool.Pool
	jwtManager     *auth.JwtManager
	passwordHasher hash.PasswordHasher
}

func NewAuthHandler(pgxPool *pgxpool.Pool, jwtManager *auth.JwtManager, passwordHasher hash.PasswordHasher) *AuthHandler {
	return &AuthHandler{
		pgxPool:        pgxPool,
		jwtManager:     jwtManager,
		passwordHasher: passwordHasher,
	}
}

type accountView struct {
	ID           string  `json:"id,omitempty"`
	RegisteredAt string  `json:"registered_at,omitempty"`
	UpdatedAt    string  `json:"updated_at,omitempty"`
	ClosedAt     *string `json:"closed_at,omitempty"`
}

func newAccountView(account *model.Account) *accountView {
	view := &accountView{
		ID:           account.ID.String(),
		RegisteredAt: account.RegisteredAt.Format(time.RFC3339),
		UpdatedAt:    account.UpdatedAt.Format(time.RFC3339),
	}
	if account.ClosedAt != nil {
		view.ClosedAt = new(string)
		*view.ClosedAt = account.ClosedAt.Format(time.RFC3339)
	}
	return view
}

type registerRequest struct {
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	Account     *accountView `json:"account"`
}

func (a *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	passwordHash, err := a.passwordHasher.Hash(req.Password)
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

	if _, err = a.pgxPool.Exec(
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

	jwt, err := a.jwtManager.New(account.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(authResponse{
		AccessToken: jwt,
		TokenType:   "Bearer",
		Account:     newAccountView(account),
	})
}
