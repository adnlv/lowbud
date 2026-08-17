-- +goose Up
CREATE TABLE ledger_transactions (
    id UUID PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE ledger_transactions;
