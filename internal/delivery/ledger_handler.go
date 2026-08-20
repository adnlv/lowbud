package delivery

import (
	"net/http"
	"strconv"
	"time"

	"github.com/adnlv/lowbud/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type LedgerHandler struct {
	DB           *pgxpool.Pool
	UUIDProvider domain.UUIDProvider
}

func NewLedgerHandler(db *pgxpool.Pool, uuidProvider domain.UUIDProvider) *LedgerHandler {
	return &LedgerHandler{
		DB:           db,
		UUIDProvider: uuidProvider,
	}
}

type balanceResponse struct {
	Amount decimal.Decimal `json:"amount"`
}

func (h *LedgerHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	const getBalanceQuery = `
SELECT COALESCE(SUM(l.amount), 0)
FROM accounts a
LEFT JOIN ledger_entries l ON l.account_id = a.id
WHERE a.id = $1 AND a.closed_at IS NULL
GROUP BY a.id;
`
	var balance decimal.Decimal
	if err := h.DB.QueryRow(r.Context(), getBalanceQuery, accessTokenClaims.AccountID).Scan(&balance); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, &balanceResponse{Amount: balance})
}

type transactionHistoryItem struct {
	ID          string          `db:"id" json:"id"`
	Description string          `db:"description" json:"description"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	Amount      decimal.Decimal `db:"amount" json:"amount"`
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

	const transactionHistoryQuery = `
SELECT 
    t.id,
    t.description,
    t.created_at,
    e.amount
FROM ledger_entries e
JOIN ledger_transactions t ON t.id = e.ledger_transaction_id
WHERE e.account_id = $1
ORDER BY e.created_at DESC
LIMIT $2
OFFSET $3
`
	offset := (page - 1) * perPage
	rows, err := h.DB.Query(r.Context(), transactionHistoryQuery, accessTokenClaims.AccountID, perPage, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[transactionHistoryItem])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, items)
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
	if err := h.UUIDProvider.ValidateUUID(transactionId); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	const getTransactionQuery = `
SELECT 
    t.id,
    t.description,
    t.created_at,
    e.amount
FROM ledger_transactions t
JOIN ledger_entries e ON e.ledger_transaction_id = t.id
WHERE t.id = $1 AND e.account_id = $2;
`
	var transaction domain.LedgerTransaction
	var entry domain.LedgerEntry

	if err := h.DB.QueryRow(
		r.Context(),
		getTransactionQuery,
		transactionId,
		accessTokenClaims.AccountID,
	).Scan(
		&transaction.ID,
		&transaction.Description,
		&transaction.CreatedAt,
		&entry.Amount,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusOK, &getTransactionResponse{
		ID:          transaction.ID,
		Description: transaction.Description,
		CreatedAt:   transaction.CreatedAt,
		Amount:      entry.Amount,
	})
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

	if !req.Amount.IsPositive() {
		writeError(w, http.StatusUnprocessableEntity, "negative and zero transfers are rejected")
		return
	}

	if accessTokenClaims.AccountID == req.DestinationAccountID {
		writeError(w, http.StatusUnprocessableEntity, "can't transfer funds to themself")
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	const lockSenderQuery = `
SELECT id FROM accounts WHERE id = $1 AND closed_at IS NULL FOR UPDATE
`
	var senderId string
	if err = tx.QueryRow(r.Context(), lockSenderQuery, accessTokenClaims.AccountID).Scan(&senderId); err != nil {
		writeError(w, http.StatusNotFound, "sender account not found or closed")
		return
	}

	const checkDestinationQuery = `
SELECT id FROM accounts WHERE id = $1 AND closed_at IS NULL
`
	var destinationId string
	if err = tx.QueryRow(r.Context(), checkDestinationQuery, req.DestinationAccountID).Scan(&destinationId); err != nil {
		writeError(w, http.StatusBadRequest, "destination account not found or closed")
		return
	}

	const totalBalanceQuery = `
SELECT COALESCE(SUM(amount), 0) FROM ledger_entries WHERE account_id = $1
`
	var balance decimal.Decimal
	if err = tx.QueryRow(r.Context(), totalBalanceQuery, accessTokenClaims.AccountID).Scan(&balance); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Amount.GreaterThan(balance) {
		writeError(w, http.StatusPaymentRequired, "not enough money")
		return
	}

	transactionId, err := h.UUIDProvider.NewUUID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	transaction := &domain.LedgerTransaction{
		ID:             transactionId,
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
		CreatedAt:      time.Now(),
	}

	const insertTransactionQuery = `
INSERT INTO ledger_transactions (id, idempotency_key, description, created_at) 
VALUES ($1, $2, $3, $4)
`
	if _, err = tx.Exec(
		r.Context(),
		insertTransactionQuery,
		transaction.ID,
		transaction.IdempotencyKey,
		transaction.Description,
		transaction.CreatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	creditEntryId, err := h.UUIDProvider.NewUUID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	creditEntry := &domain.LedgerEntry{
		ID:                  creditEntryId,
		LedgerTransactionID: transactionId,
		AccountID:           accessTokenClaims.AccountID,
		Amount:              req.Amount.Neg(),
		CreatedAt:           time.Now(),
	}

	debitEntryId, err := h.UUIDProvider.NewUUID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	debitEntry := &domain.LedgerEntry{
		ID:                  debitEntryId,
		LedgerTransactionID: transactionId,
		AccountID:           req.DestinationAccountID,
		Amount:              req.Amount,
		CreatedAt:           time.Now(),
	}

	const insertLedgerEntryQuery = `
INSERT INTO ledger_entries (id, ledger_transaction_id, account_id, amount, created_at) 
VALUES ($1, $2, $3, $4, $5)
`
	if _, err = tx.Exec(
		r.Context(),
		insertLedgerEntryQuery,
		creditEntry.ID,
		creditEntry.LedgerTransactionID,
		creditEntry.AccountID,
		creditEntry.Amount,
		creditEntry.CreatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err = tx.Exec(
		r.Context(),
		insertLedgerEntryQuery,
		debitEntry.ID,
		debitEntry.LedgerTransactionID,
		debitEntry.AccountID,
		debitEntry.Amount,
		debitEntry.CreatedAt,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeStatusCode(w, http.StatusCreated)
}
