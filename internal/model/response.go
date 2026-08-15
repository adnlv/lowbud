package model

type ApiErrorResponse struct {
	Code    int
	Message string
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	Account      *AccountView `json:"account"`
}
