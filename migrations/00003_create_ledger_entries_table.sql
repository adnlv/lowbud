-- +goose Up
CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY,
    ledger_transaction_id UUID NOT NULL REFERENCES ledger_transactions (id) ON DELETE RESTRICT,
    account_id UUID NOT NULL REFERENCES accounts (id),
    amount NUMERIC(16, 4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_entries_account_created ON ledger_entries (account_id, created_at DESC);

-- +goose Down
DROP INDEX idx_ledger_entries_account_created;
DROP TABLE ledger_entries;
