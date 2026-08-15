package delivery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/adnlv/lowbud/internal/model"
)

func writeJson(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJson(w, status, model.ApiErrorResponse{
		Code:    status,
		Message: message,
	})
}

func decodeAndValidateRequest(body io.ReadCloser, req any) error {
	if err := json.NewDecoder(body).Decode(req); err != nil {
		return fmt.Errorf("decodeAndValidateRequest: failed to decode request: %v", err)
	}
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("decodeAndValidateRequest: failed to validate request: %v", err)
	}
	return nil
}
