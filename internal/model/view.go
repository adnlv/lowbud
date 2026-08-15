package model

import "time"

type AccountView struct {
	ID           string  `json:"id,omitempty"`
	RegisteredAt string  `json:"registered_at,omitempty"`
	UpdatedAt    string  `json:"updated_at,omitempty"`
	ClosedAt     *string `json:"closed_at,omitempty"`
}

func NewAccountView(account *Account) *AccountView {
	view := &AccountView{
		ID:           account.ID.String(),
		RegisteredAt: account.RegisteredAt.Format(time.RFC3339),
		UpdatedAt:    account.UpdatedAt.Format(time.RFC3339),
	}
	if account.ClosedAt != nil {
		closedAtStr := account.ClosedAt.Format(time.RFC3339)
		view.ClosedAt = &closedAtStr
	}
	return view
}
