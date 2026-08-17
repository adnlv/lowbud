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

type LedgerTransaction struct {
	ID                   string
	SourceAccountID      string
	DestinationAccountID string
	Amount               uint64
	CreatedAt            time.Time
}
