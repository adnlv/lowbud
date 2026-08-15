package auth

import "github.com/google/uuid"

type AccessTokenPayload struct {
	AccountIDs []uuid.UUID `json:"account_ids"`
}

func NewAccountIDsList(ids ...uuid.UUID) []uuid.UUID {
	l := make([]uuid.UUID, 0, len(ids))
	l = append(l, ids...)
	return l
}

type TokenManager interface {
	GenerateAccessToken(payload *AccessTokenPayload) (string, error)
	ParseAccessToken(tokenStr string) (*AccessTokenPayload, error)
	GenerateRefreshToken() (string, error)
	ValidateRefreshToken(tokenStr string) error
}
