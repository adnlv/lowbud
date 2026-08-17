package delivery

import (
	"encoding/json"
	"io"
	"net/http"

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
