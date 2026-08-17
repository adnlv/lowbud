package domain

import (
	"time"

	"github.com/shopspring/decimal"
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

type LedgerTransaction struct {
	ID             string
	IdempotencyKey string
	Description    string
	CreatedAt      time.Time
}

type LedgerEntry struct {
	ID                  string
	LedgerTransactionID string
	AccountID           string
	Amount              decimal.Decimal
	CreatedAt           time.Time
}

type AccessTokenClaims struct {
	AccountID string `json:"account_id"`
}
