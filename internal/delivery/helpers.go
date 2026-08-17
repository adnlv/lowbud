package delivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/adnlv/lowbud/internal/domain"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type APIResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func writeStatusCode(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
}

func writeJson(w http.ResponseWriter, statusCode int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJson(w, statusCode, APIResponse{
		Code:    statusCode,
		Message: message,
	})
}

func decodeAndValidateJson(body io.ReadCloser, data any) error {
	if err := json.NewDecoder(body).Decode(data); err != nil {
		return err
	}
	if err := validate.Struct(data); err != nil {
		return err
	}
	return nil
}

type ctxKey string

const accessTokenClaimsContextKey ctxKey = "access_token_claims"

func accessTokenClaimsFromContext(ctx context.Context) *domain.AccessTokenClaims {
	claims, _ := ctx.Value(accessTokenClaimsContextKey).(*domain.AccessTokenClaims)
	return claims
}
