package delivery

import (
	"net/http"
	"time"

	"github.com/adnlv/lowbud/internal/domain"
)

type AccountHandler struct {
	AccountService domain.AccountService
}

func NewAccountHandler(accountService domain.AccountService) *AccountHandler {
	return &AccountHandler{AccountService: accountService}
}

type accountJsonView struct {
	Email        string `json:"email,omitempty"`
	Forename     string `json:"forename,omitempty"`
	Surname      string `json:"surname,omitempty"`
	RegisteredAt string `json:"registered_at,omitempty"`
}

func newAccountJSONView(account *domain.GetAccountInfoResult) *accountJsonView {
	return &accountJsonView{
		Email:        account.Email,
		Forename:     account.Forename,
		Surname:      account.Surname,
		RegisteredAt: account.RegisteredAt.Format(time.RFC3339),
	}
}

func (h *AccountHandler) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	result, err := h.AccountService.GetAccountInfo(r.Context(), &domain.GetAccountInfoQuery{AccountID: accessTokenClaims.AccountID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, newAccountJSONView(result))
}

func (h *AccountHandler) CloseAccount(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	if err := h.AccountService.CloseAccount(r.Context(), &domain.CloseAccountCommand{AccountID: accessTokenClaims.AccountID}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeStatusCode(w, http.StatusOK)
}
