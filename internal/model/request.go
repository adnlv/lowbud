package model

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type RegisterRequest struct {
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	AccountID string `json:"account_id" validate:"required,uuid"`
	Password  string `json:"password" validate:"required"`
}

type DebitRequest struct {
	DestinationAccountID uuid.UUID       `json:"destination_account_id" validate:"required,uuid"`
	Amount               decimal.Decimal `json:"amount" validate:"required"`
}
