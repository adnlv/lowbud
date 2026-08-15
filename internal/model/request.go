package model

type RegisterRequest struct {
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	AccountID string `json:"account_id" validate:"required,uuid"`
	Password  string `json:"password" validate:"required"`
}
