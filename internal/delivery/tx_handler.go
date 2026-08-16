package delivery

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/adnlv/lowbud/internal/model"
	uuid2 "github.com/adnlv/lowbud/pkg/uuid"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type sqlConn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txDbView struct {
	ID                   pgtype.UUID        `db:"id"`
	SourceAccountID      pgtype.UUID        `db:"source_account_id"`
	DestinationAccountID pgtype.UUID        `db:"destination_account_id"`
	Amount               pgtype.Numeric     `db:"amount"`
	CreatedAt            pgtype.Timestamptz `db:"created_at"`
}

func newDbViewFromTx(tx *model.Transaction) *txDbView {
	return &txDbView{
		ID: pgtype.UUID{
			Bytes: tx.ID,
			Valid: true,
		},
		SourceAccountID: pgtype.UUID{
			Bytes: tx.SourceAccountID,
			Valid: true,
		},
		DestinationAccountID: pgtype.UUID{
			Bytes: tx.DestinationAccountID,
			Valid: true,
		},
		Amount: pgtype.Numeric{
			Int:   tx.Amount.Coefficient(),
			Exp:   tx.Amount.Exponent(),
			Valid: true,
		},
		CreatedAt: pgtype.Timestamptz{
			Time:  tx.CreatedAt,
			Valid: true,
		},
	}
}

func newTxFromDbView(view *txDbView) *model.Transaction {
	return &model.Transaction{
		ID:                   view.ID.Bytes,
		SourceAccountID:      view.SourceAccountID.Bytes,
		DestinationAccountID: view.DestinationAccountID.Bytes,
		Amount:               decimal.New(view.Amount.Int.Int64(), view.Amount.Exp),
		CreatedAt:            view.CreatedAt.Time,
	}
}

func collectTransactions(conn sqlConn, ctx context.Context, query string, args ...any) ([]model.Transaction, error) {
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rows: %v", err)
	}
	defer rows.Close()

	txDbViews, err := pgx.CollectRows(rows, pgx.RowToStructByName[txDbView])
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %v", err)
	}

	transactions := make([]model.Transaction, 0, len(txDbViews))
	for _, view := range txDbViews {
		transactions = append(transactions, *newTxFromDbView(&view))
	}
	return transactions, nil
}

func collectAllDebits(conn sqlConn, ctx context.Context, destinationAccountID uuid.UUID) ([]model.Transaction, error) {
	const q = `SELECT * FROM transactions WHERE destination_account_id = $1 FOR NO KEY UPDATE`
	return collectTransactions(conn, ctx, q, destinationAccountID)
}

func collectAllCredits(conn sqlConn, ctx context.Context, sourceAccountId uuid.UUID) ([]model.Transaction, error) {
	const q = `SELECT * FROM transactions WHERE source_account_id = $1 FOR NO KEY UPDATE`
	return collectTransactions(conn, ctx, q, sourceAccountId)
}

func insertTx(conn sqlConn, ctx context.Context, transaction *model.Transaction) error {
	dbView := newDbViewFromTx(transaction)
	const q = `insert into transactions(id, source_account_id, destination_account_id, amount, created_at) values ($1, $2, $3, $4, $5)`
	if _, err := conn.Exec(
		ctx,
		q,
		dbView.ID,
		dbView.SourceAccountID,
		dbView.DestinationAccountID,
		dbView.Amount,
		dbView.CreatedAt,
	); err != nil {
		return fmt.Errorf("failed to insert row: %v", err.Error())
	}
	return nil
}

type TxHandler struct {
	pgxPool       *pgxpool.Pool
	uuidGenerator uuid2.Generator
}

func NewTxHandler(pgxPool *pgxpool.Pool, uuidGenerator uuid2.Generator) *TxHandler {
	return &TxHandler{
		pgxPool:       pgxPool,
		uuidGenerator: uuidGenerator,
	}
}

func (h *TxHandler) Debit(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())

	var req model.DebitRequest
	if err := decodeAndValidateRequest(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.pgxPool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	debits, err := collectAllDebits(tx, r.Context(), claims.AccountID)
	if err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error())
		return
	}

	credits, err := collectAllCredits(tx, r.Context(), claims.AccountID)
	if err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error())
		return
	}

	sum := decimal.Zero
	for _, debit := range debits {
		sum.Add(debit.Amount)
	}
	for _, credit := range credits {
		sum.Sub(credit.Amount)
	}

	if req.Amount.GreaterThan(sum) {
		writeError(w, http.StatusPaymentRequired, "not enough money")
		return
	}

	bankTxId, err := h.uuidGenerator.Generate()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = insertTx(tx, r.Context(), &model.Transaction{
		ID:                   bankTxId,
		SourceAccountID:      claims.AccountID,
		DestinationAccountID: req.DestinationAccountID,
		Amount:               req.Amount,
		CreatedAt:            time.Now(),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, http.StatusCreated, nil)
}
