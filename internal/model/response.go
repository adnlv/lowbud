package model

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	Account     *AccountView `json:"account"`
}
