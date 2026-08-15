package model

import "github.com/google/uuid"

type JwtClaims struct {
	AccountID uuid.UUID `json:"account_id"`
}
