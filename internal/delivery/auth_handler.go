package delivery

import (
	"context"
	"net/http"
	"strings"

	"github.com/adnlv/lowbud/internal/domain"
)

type AuthHandler struct {
	AuthService domain.AuthService
}

func NewAuthHandler(authService domain.AuthService) *AuthHandler {
	return &AuthHandler{AuthService: authService}
}

type registerAccountRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Forename string `json:"forename" validate:"required"`
	Surname  string `json:"surname" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (h *AuthHandler) RegisterAccount(w http.ResponseWriter, r *http.Request) {
	req := new(registerAccountRequest)
	if err := decodeAndValidateJson(r.Body, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.AuthService.RegisterAccount(r.Context(), &domain.RegisterAccountCommand{
		Email:    req.Email,
		Forename: req.Forename,
		Surname:  req.Surname,
		Password: req.Password,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeStatusCode(w, http.StatusCreated)
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type authResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) BasicLogin(w http.ResponseWriter, r *http.Request) {
	req := new(loginRequest)
	if err := decodeAndValidateJson(r.Body, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.AuthService.BasicLogin(r.Context(), &domain.BasicLoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, &authResponse{AccessToken: result.AccessToken})
}

func (h *AuthHandler) DemandAccessTokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "authorization header is missing")
		}

		authHeaderParts := strings.SplitN(authHeader, " ", 2)
		if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "authorization header is malformed")
			return
		}

		claims, err := h.AuthService.ParseAccessToken(r.Context(), authHeaderParts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), accessTokenClaimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}
