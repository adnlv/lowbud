package delivery

import (
	"net/http"
	"strconv"
	"time"

	"github.com/adnlv/lowbud/internal/domain"
	"github.com/shopspring/decimal"
)

type LedgerHandler struct {
	LedgerService domain.LedgerService
}

func NewLedgerHandler(ledgerService domain.LedgerService) *LedgerHandler {
	return &LedgerHandler{LedgerService: ledgerService}
}

type getBalanceResponse struct {
	Amount decimal.Decimal `json:"amount"`
}

func (h *LedgerHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	result, err := h.LedgerService.GetBalance(r.Context(), &domain.GetBalanceQuery{AccountID: accessTokenClaims.AccountID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, &getBalanceResponse{Amount: result.Amount})
}

type getTransactionResponse struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
	Amount      decimal.Decimal `json:"amount"`
}

func (h *LedgerHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	transactionId := r.PathValue("transactionId")
	result, err := h.LedgerService.GetTransaction(r.Context(), &domain.GetTransactionQuery{
		AccountID:     accessTokenClaims.AccountID,
		TransactionID: transactionId,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, &getTransactionResponse{
		ID:          result.ID,
		Description: result.Description,
		CreatedAt:   result.CreatedAt,
		Amount:      result.Amount,
	})
}

func (h *LedgerHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	queryValues := r.URL.Query()

	var page uint64 = 1
	const pageKey = "page"
	if queryValues.Has(pageKey) {
		parseUint, err := strconv.ParseUint(queryValues.Get(pageKey), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		page = parseUint
	}
	if page == 0 {
		writeError(w, http.StatusBadRequest, "page must be a positive number")
		return
	}

	var perPage uint64 = 100
	const perPageKey = "per_page"
	if queryValues.Has(perPageKey) {
		parseUint, err := strconv.ParseUint(queryValues.Get(perPageKey), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		perPage = parseUint
	}
	if perPage > 100 {
		writeError(w, http.StatusBadRequest, "per_page is too high")
		return
	}

	items, err := h.LedgerService.GetTransactionHistory(r.Context(), &domain.GetTransactionHistoryQuery{
		Page:      page,
		PerPage:   perPage,
		AccountID: accessTokenClaims.AccountID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, items)
}

type ledgerTransferRequest struct {
	IdempotencyKey       string          `json:"idempotency_key" validate:"required"`
	Description          string          `json:"description"`
	DestinationAccountID string          `json:"destination_account_id" validate:"required,uuid"`
	Amount               decimal.Decimal `json:"amount" validate:"required"`
}

func (h *LedgerHandler) TransferFunds(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	req := new(ledgerTransferRequest)
	if err := decodeAndValidateJson(r.Body, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.LedgerService.TransferFunds(r.Context(), &domain.TransferFundsCommand{
		IdempotencyKey:       req.IdempotencyKey,
		Description:          req.Description,
		SourceAccountID:      accessTokenClaims.AccountID,
		DestinationAccountID: req.DestinationAccountID,
		Amount:               req.Amount,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeStatusCode(w, http.StatusCreated)
}
