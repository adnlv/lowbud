package delivery

import (
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

	// TODO: store session information in the database
	refreshToken, err := h.tokenManager.GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusCreated, model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Account:      model.NewAccountView(account),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeAndValidateRequest(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var err error
	var account model.Account
	account.ID, err = h.uuidGenerator.Parse(req.AccountID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = h.pgxPool.QueryRow(
		r.Context(),
		`SELECT password_hash, registered_at, updated_at, closed_at FROM accounts WHERE id = $1`,
		account.ID,
	).Scan(
		&account.PasswordHash,
		&account.RegisteredAt,
		&account.UpdatedAt,
		&account.ClosedAt,
	); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err = h.passwordHasher.Compare(req.Password, account.PasswordHash); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	accessTokenPayload := &auth.AccessTokenPayload{AccountID: account.ID}
	accessToken, err := h.tokenManager.GenerateAccessToken(accessTokenPayload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// TODO: store session information in the database
	refreshToken, err := h.tokenManager.GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		Account:      model.NewAccountView(&account),
	})
}
