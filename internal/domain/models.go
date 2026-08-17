package domain

import (
	"time"
)

type Account struct {
	ID           string
	Forename     string
	Surname      string
	Email        string
	PasswordHash string
	RegisteredAt time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

type LedgerTransaction struct {
	ID                   string
	SourceAccountID      string
	DestinationAccountID string
	Amount               uint64
	CreatedAt            time.Time
}
