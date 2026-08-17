package domain

import (
	"time"
)

type Account struct {
	ID           string
	Email        string
	Forename     string
	Surname      string
	PasswordHash string
	RegisteredAt time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

type AccessTokenClaims struct {
	AccountID string `json:"account_id"`
}
