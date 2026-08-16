package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Account struct {
	ID           uuid.UUID
	PasswordHash string
	RegisteredAt time.Time
	UpdatedAt    time.Time
	ClosedAt     *time.Time
}

type Transaction struct {
	ID                 uuid.UUID
	SourceAccount      uuid.UUID
	DestinationAccount uuid.UUID
	Amount             decimal.Decimal
	CreatedAt          time.Time
}
