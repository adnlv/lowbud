package delivery

import (
	"net/http"
	"time"

	"github.com/adnlv/lowbud/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountHandler struct {
	DB *pgxpool.Pool
}

type AccountJSONView struct {
	ID           string `json:"id,omitempty"`
	Email        string `json:"email,omitempty"`
	Forename     string `json:"forename,omitempty"`
	Surname      string `json:"surname,omitempty"`
	RegisteredAt string `json:"registered_at"`
}

func NewAccountJSONView(account *domain.Account) *AccountJSONView {
	return &AccountJSONView{
		ID:           account.ID,
		Email:        account.Email,
		Forename:     account.Forename,
		Surname:      account.Surname,
		RegisteredAt: account.RegisteredAt.Format(time.RFC3339),
	}
}

func (h *AccountHandler) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	account := &domain.Account{ID: accessTokenClaims.AccountID}

	const getAccountByIdQuery = `
SELECT email, forename, surname, registered_at FROM accounts WHERE id = $1 AND closed_at IS NULL
`
	if err := h.DB.QueryRow(
		r.Context(),
		getAccountByIdQuery,
		account.ID,
	).Scan(
		&account.Email,
		&account.Forename,
		&account.Surname,
		&account.RegisteredAt,
	); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJson(w, http.StatusOK, NewAccountJSONView(account))
}

func (h *AccountHandler) CloseAccount(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	account := &domain.Account{
		ID:       accessTokenClaims.AccountID,
		ClosedAt: new(time.Now()),
	}

	const closeAccountByIdQuery = `
UPDATE accounts SET closed_at = $2 WHERE id = $1 
`
	if _, err := h.DB.Exec(
		r.Context(),
		closeAccountByIdQuery,
		account.ID,
		account.ClosedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeStatusCode(w, http.StatusOK)
}

func NewAccountHandler(db *pgxpool.Pool) *AccountHandler {
	return &AccountHandler{DB: db}
}
