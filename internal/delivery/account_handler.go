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
