package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID           string
	PasswordHash string
	RegisteredAt time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

type Transaction struct {
	ID                 string
	SourceAccount      string
	DestinationAccount string
	CurrencyCode       string
	Amount             decimal.Decimal
	CreatedAt          time.Time
}
