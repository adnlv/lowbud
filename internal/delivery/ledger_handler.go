package delivery

import (
	"net/http"
	"time"

	"github.com/adnlv/lowbud/internal/domain"
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

type createLedgerTransactionRequest struct {
	IdempotencyKey       string          `json:"idempotency_key" validate:"required"`
	Description          string          `json:"description"`
	DestinationAccountID string          `json:"destination_account_id" validate:"required,uuid"`
	Amount               decimal.Decimal `json:"amount" validate:"required"`
}

func (h *LedgerHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	accessTokenClaims := accessTokenClaimsFromContext(r.Context())

	req := new(createLedgerTransactionRequest)
	if err := decodeAndValidateJson(r.Body, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !req.Amount.IsPositive() {
		writeError(w, http.StatusBadRequest, "negative and zero transfers are rejected")
		return
	}

	if accessTokenClaims.AccountID == req.DestinationAccountID {
		writeError(w, http.StatusUnprocessableEntity, "can't transfer funds to themself")
		return
	}

	const totalBalanceQuery = `
SELECT COALESCE(SUM(le.amount), 0) AS total_balance
FROM accounts a LEFT JOIN ledger_entries le ON le.account_id = a.id
WHERE a.id = $1 AND a.closed_at IS NULL
GROUP BY a.id
`
	balance := decimal.Zero
	if err := h.DB.QueryRow(r.Context(), totalBalanceQuery, accessTokenClaims.AccountID).Scan(&balance); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Amount.GreaterThan(balance) {
		writeError(w, http.StatusPaymentRequired, "not enough money")
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

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
